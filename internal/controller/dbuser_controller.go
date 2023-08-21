/*
 * Copyright 2023 Nikolai Rodionov (allanger)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kindav1beta1 "github.com/db-operator/db-operator/api/v1beta1"
	"github.com/db-operator/db-operator/internal/utils/templates"
	"github.com/db-operator/db-operator/pkg/utils/database"
	"github.com/db-operator/db-operator/pkg/utils/kci"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
)

// DbUserReconciler reconciles a DbUser object
type DbUserReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Interval     time.Duration
	Log          logr.Logger
	Recorder     record.EventRecorder
	CheckChanges bool
}

//+kubebuilder:rbac:groups=kinda.rocks,resources=dbusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kinda.rocks,resources=dbusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kinda.rocks,resources=dbusers/finalizers,verbs=update

func (r *DbUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("dbuser", req.NamespacedName)

	reconcilePeriod := r.Interval * time.Second
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	dbucr := &kindav1beta1.DbUser{}
	err := r.Get(ctx, req.NamespacedName, dbucr)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Requested object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcileResult, nil
		}
		// Error reading the object - requeue the request.
		return reconcileResult, err
	}

	// Update object status always when function exit abnormally or through a panic.
	defer func() {
		if err := r.Status().Update(ctx, dbucr); err != nil {
			logrus.Errorf("failed to update status - %s", err)
		}
	}()

	// Get the DB by the reference provided in the manifest
	dbcr := &kindav1beta1.Database{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: dbucr.Spec.DatabaseRef}, dbcr); err != nil {
		return r.manageError(ctx, dbucr, err, false)
	}

	// Check if DbUser is marked to be deleted
	ownership := []metav1.OwnerReference{}
	ownership = append(ownership, metav1.OwnerReference{
		APIVersion: dbucr.APIVersion,
		Kind:       dbucr.Kind,
		Name:       dbucr.Name,
		UID:        dbucr.GetUID(),
	},
	)

	engine, err := dbcr.GetEngineType()
	if err != nil {
		return r.manageError(ctx, dbucr, err, false)
	}

	userSecret, err := r.getDbUserSecret(ctx, dbucr)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			dbName := fmt.Sprintf("%s-%s", dbucr.Namespace, dbucr.Spec.DatabaseRef)
			secretData, err := generateDatabaseSecretData(dbucr.ObjectMeta, engine, dbName)
			if err != nil {
				logrus.Errorf("can not generate credentials for database - %s", err)
				return r.manageError(ctx, dbucr, err, false)
			}
			newDbUserSecret := kci.SecretBuilder(dbucr.Spec.SecretName, dbucr.Namespace, secretData, ownership)
			err = r.Create(ctx, newDbUserSecret)
			if err != nil {
				// failed to create secret
				return r.manageError(ctx, dbucr, err, false)
			}
			userSecret = newDbUserSecret
		} else {
			logrus.Errorf("could not get database secret - %s", err)
			return r.manageError(ctx, dbucr, err, true)
		}
	}

	creds, err := parseDbUserSecretData(dbcr.Status.InstanceRef.Spec.Engine, userSecret.Data)
	if err != nil {
		return r.manageError(ctx, dbucr, err, false)
	}

	// If we don't check for changes, status should be false on each reconciliation
	if !r.CheckChanges || isDbUserChanged(dbucr, userSecret) {
		dbucr.Status.Status = false
	}

	db, dbuser, err := determinDatabaseType(dbcr, creds)
	if err != nil {
		// failed to determine database type
		return r.manageError(ctx, dbucr, err, false)
	}
	adminSecretResource, err := r.getAdminSecret(ctx, dbcr)
	if err != nil {
		// failed to get admin secret
		return r.manageError(ctx, dbucr, err, false)
	}

	adminCred, err := db.ParseAdminCredentials(adminSecretResource.Data)
	if err != nil {
		// failed to parse database admin secret
		return r.manageError(ctx, dbucr, err, false)
	}

	dbuser.AccessType = dbucr.Spec.AccessType
	dbuser.Password = creds.Password
	dbuser.Username = fmt.Sprintf("%s-%s", dbucr.GetObjectMeta().GetNamespace(), dbucr.GetObjectMeta().GetName())

	if dbucr.GetDeletionTimestamp() != nil {
		if containsString(dbucr.ObjectMeta.Finalizers, "dbuser."+dbucr.Name) {
			if err := database.DeleteUser(db, dbuser, adminCred); err != nil {
				logrus.Errorf("DBUser: namespace=%s, name=%s failed deleting a user - %s", dbucr.Namespace, dbucr.Name, err)
				return r.manageError(ctx, dbucr, err, false)
			}
			kci.RemoveFinalizer(&dbucr.ObjectMeta, "dbuser."+dbucr.Name)
			err = r.Update(ctx, dbucr)
			if err != nil {
				logrus.Errorf("error resource updating - %s", err)
				return r.manageError(ctx, dbucr, err, false)
			}
			kci.RemoveFinalizer(&dbcr.ObjectMeta, "dbuser."+dbucr.Name)
			err = r.Update(ctx, dbcr)
			if err != nil {
				logrus.Errorf("error resource updating - %s", err)
				return r.manageError(ctx, dbucr, err, false)
			}

		}
	} else {
		if !dbcr.Status.Status {
			err := fmt.Errorf("database %s is not ready yet", dbcr.Name)
			return r.manageError(ctx, dbucr, err, true)
		}

		//Init the DbUser struct depending on a type
		if !dbucr.Status.Status {
			if !dbucr.Status.Created {
				r.Log.Info(fmt.Sprintf("creating a user: %s", dbucr.GetObjectMeta().GetName()))
				if err := database.CreateUser(db, dbuser, adminCred); err != nil {
					return r.manageError(ctx, dbucr, err, false)
				}
				kci.AddFinalizer(&dbucr.ObjectMeta, "dbuser."+dbucr.Name)
				err = r.Update(ctx, dbucr)
				if err != nil {
					logrus.Errorf("error resource updating - %s", err)
					return r.manageError(ctx, dbucr, err, false)
				}
				kci.AddFinalizer(&dbcr.ObjectMeta, "dbuser."+dbucr.Name)
				err = r.Update(ctx, dbcr)
				if err != nil {
					logrus.Errorf("error resource updating - %s", err)
					return r.manageError(ctx, dbucr, err, false)
				}
				dbucr.Status.Created = true
			} else {
				r.Log.Info(fmt.Sprintf("updating a user %s", dbucr.GetObjectMeta().GetName()))
				if err := database.UpdateUser(db, dbuser, adminCred); err != nil {
					return r.manageError(ctx, dbucr, err, false)
				}
			}
			dbucr.Status.Status = true
			dbucr.Status.DatabaseName = dbucr.Spec.DatabaseRef
		}
	}

	// Render secret templates for the user
	if err := r.renderTemplates(ctx, dbucr, dbcr); err != nil {
		return r.manageError(ctx, dbucr, err, false)
	}

	return reconcileResult, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DbUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kindav1beta1.DbUser{}).
		Complete(r)
}

func isDbUserChanged(dbucr *kindav1beta1.DbUser, userSecret *corev1.Secret) bool {
	annotations := dbucr.ObjectMeta.GetAnnotations()

	return annotations["checksum/spec"] != kci.GenerateChecksum(dbucr.Spec) ||
		annotations["checksum/secret"] != generateChecksumSecretValue(userSecret)
}

func (r *DbUserReconciler) getDbUserSecret(ctx context.Context, dbucr *kindav1beta1.DbUser) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	key := types.NamespacedName{
		Namespace: dbucr.Namespace,
		Name:      dbucr.Spec.SecretName,
	}
	err := r.Get(ctx, key, secret)
	if err != nil {
		return nil, err
	}

	return secret, nil

}

func (r *DbUserReconciler) manageError(ctx context.Context, dbucr *kindav1beta1.DbUser, issue error, requeue bool) (reconcile.Result, error) {
	dbucr.Status.Status = false
	logrus.Errorf("DB: namespace=%s, name=%s failed - %s", dbucr.Namespace, dbucr.Name, issue)

	retryInterval := 60 * time.Second

	r.Recorder.Event(dbucr, "Warning", "Failed", issue.Error())
	err := r.Status().Update(ctx, dbucr)
	if err != nil {
		logrus.Error(err, "unable to update status")
		return reconcile.Result{
			RequeueAfter: retryInterval,
			Requeue:      requeue,
		}, nil
	}

	// TODO: implementing reschedule calculation based on last updated time
	return reconcile.Result{
		RequeueAfter: retryInterval,
		Requeue:      requeue,
	}, nil
}

func parseDbUserSecretData(engine string, data map[string][]byte) (database.Credentials, error) {
	cred := database.Credentials{}

	switch engine {
	case "postgres":
		if name, ok := data["POSTGRES_DB"]; ok {
			cred.Name = string(name)
		} else {
			return cred, errors.New("POSTGRES_DB key does not exist in secret data")
		}

		if user, ok := data["POSTGRES_USER"]; ok {
			cred.Username = string(user)
		} else {
			return cred, errors.New("POSTGRES_USER key does not exist in secret data")
		}

		if pass, ok := data["POSTGRES_PASSWORD"]; ok {
			cred.Password = string(pass)
		} else {
			return cred, errors.New("POSTGRES_PASSWORD key does not exist in secret data")
		}

		return cred, nil
	case "mysql":
		if name, ok := data["DB"]; ok {
			cred.Name = string(name)
		} else {
			return cred, errors.New("DB key does not exist in secret data")
		}

		if user, ok := data["USER"]; ok {
			cred.Username = string(user)
		} else {
			return cred, errors.New("USER key does not exist in secret data")
		}

		if pass, ok := data["PASSWORD"]; ok {
			cred.Password = string(pass)
		} else {
			return cred, errors.New("PASSWORD key does not exist in secret data")
		}

		return cred, nil
	default:
		return cred, errors.New("not supported engine type")
	}
}

func (r *DbUserReconciler) getAdminSecret(ctx context.Context, dbcr *kindav1beta1.Database) (*corev1.Secret, error) {
	instance, err := dbcr.GetInstanceRef()
	if err != nil {
		// failed to get DbInstanceRef this case should not happen
		return nil, err
	}

	// get database admin credentials
	secret := &corev1.Secret{}

	err = r.Get(ctx, instance.Spec.AdminUserSecret.ToKubernetesType(), secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

func (r *DbUserReconciler) renderTemplates(ctx context.Context, dbuser *kindav1beta1.DbUser, dbcr *kindav1beta1.Database) error {
	dbuserSecret, err := r.getDbUserSecret(ctx, dbuser)
	if err != nil {
		return err
	}

	dbuserConfigMap, err := r.getDbUserCongigMap(ctx, dbuser)
	if err != nil {
		return err
	}
	engine, err := dbcr.GetEngineType()
	if err != nil {
		return err
	}
	creds, err := parseDbUserSecretData(engine, dbuserSecret.Data)
	if err != nil {
		return err
	}

	db, _, err := determinDatabaseType(dbcr, creds)
	if err != nil {
		return err
	}

	templateds, err := templates.NewTemplateDataSource(dbcr, dbuser, dbuserSecret, dbuserConfigMap, db, nil)
	if err != nil {
		return err
	}

	if err := templateds.BuildVars(dbcr.Spec.Templates); err != nil {
		return err
	}

	return nil
}

func (r *DbUserReconciler) getDbUserCongigMap(ctx context.Context, dbcr *kindav1beta1.DbUser) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	key := types.NamespacedName{
		Namespace: dbcr.Namespace,
		Name:      dbcr.Spec.SecretName,
	}
	err := r.Get(ctx, key, configMap)
	if err != nil {
		return nil, err
	}

	return configMap, nil
}

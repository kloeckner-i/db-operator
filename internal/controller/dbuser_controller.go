/*
 * Copyright 2023 DB-Operator Authors
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

package controller

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	kindav1beta1 "github.com/db-operator/db-operator/v2/api/v1beta1"
	commonhelper "github.com/db-operator/db-operator/v2/internal/helpers/common"
	dbhelper "github.com/db-operator/db-operator/v2/internal/helpers/database"
	kubehelper "github.com/db-operator/db-operator/v2/internal/helpers/kube"
	"github.com/db-operator/db-operator/v2/internal/utils/templates"
	"github.com/db-operator/db-operator/v2/pkg/consts"
	"github.com/db-operator/db-operator/v2/pkg/utils/database"
	"github.com/db-operator/db-operator/v2/pkg/utils/kci"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// DbUserReconciler reconciles a DbUser object
type DbUserReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Interval     time.Duration
	Recorder     events.EventRecorder
	CheckChanges bool
	kubeHelper   *kubehelper.KubeHelper
}

// +kubebuilder:rbac:groups=kinda.rocks,resources=dbusers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kinda.rocks,resources=dbusers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kinda.rocks,resources=dbusers/finalizers,verbs=update

// Reconcile a DbUser object
func (r *DbUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	reconcilePeriod := r.Interval
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	dbusercr := &kindav1beta1.DbUser{}
	err := r.Get(ctx, req.NamespacedName, dbusercr)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Requested object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcileResult, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Couldn't get a DbUser resource")
		return reconcileResult, err
	}

	// Update object status always when function exit abnormally or through a panic.
	defer func() {
		if err := r.Status().Update(ctx, dbusercr); err != nil {
			log.Error(err, "Failed to update status")
		}
	}()

	// Init the kubehelper object
	r.kubeHelper = kubehelper.NewKubeHelper(r.Client, r.Recorder, dbusercr)

	// Get the DB by the reference provided in the manifest
	dbcr := &kindav1beta1.Database{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: dbusercr.Spec.DatabaseRef}, dbcr); err != nil {
		return r.manageError(ctx, dbusercr, err, false)
	}

	// The secret is required for all kinds of the events, because it's used a storage for the
	// dbuser data. So even when a dbuser is removed, we need to get the secret in order to
	// let the db-operator know which exactly user must be removed.
	userSecret, err := r.getDbUserSecret(ctx, dbusercr)
	if err != nil {
		// If a secret is not found on the "delete" event it should be a critical unrecoverable
		// error, cause we don't know which user must be removed
		if k8serrors.IsNotFound(err) && !dbusercr.IsDeleted() {
			dbName := fmt.Sprintf("%s-%s", dbusercr.Namespace, dbusercr.Spec.DatabaseRef)
			secretData, err := dbhelper.GenerateDatabaseSecretData(dbusercr.ObjectMeta, dbcr.Status.Engine, dbName, dbusercr.Spec.ExistingUser)
			if err != nil {
				log.Error(err, "Could not generate credentials for database")
				return r.manageError(ctx, dbusercr, err, false)
			}
			userSecret = kci.SecretBuilder(dbusercr.Spec.SecretName, dbusercr.Namespace, secretData)
		} else {
			log.Error(err, "Could not get database secret")
			return r.manageError(ctx, dbusercr, err, true)
		}
	}

	// Apply extra metadata from the DbUser credentials spec to the
	// user credentials Secret before it is created or updated. This
	// allows users to configure labels and annotations that are
	// required by external controllers such as secret reflectors.
	if dbusercr.Spec.Credentials.Metadata != nil {
		meta := dbusercr.Spec.Credentials.Metadata
		if len(meta.ExtraLabels) > 0 {
			if userSecret.Labels == nil {
				userSecret.Labels = map[string]string{}
			}
			for k, v := range meta.ExtraLabels {
				userSecret.Labels[k] = v
			}
		}
		if len(meta.ExtraAnnotations) > 0 {
			if userSecret.Annotations == nil {
				userSecret.Annotations = map[string]string{}
			}
			for k, v := range meta.ExtraAnnotations {
				userSecret.Annotations[k] = v
			}
		}
	}

	// Make sure the secret is reflecting the actual desired state
	err = r.kubeHelper.HandleCreateOrUpdate(ctx, userSecret)
	if err != nil {
		// failed to create secret
		return r.manageError(ctx, dbusercr, err, false)
	}

	creds, err := parseDbUserSecretData(dbcr.Status.Engine, userSecret.Data)
	if err != nil {
		return r.manageError(ctx, dbusercr, err, false)
	}

	// If we don't check for changes, status should be false on each reconciliation
	if !r.CheckChanges || isDbUserChanged(dbusercr, userSecret) {
		dbusercr.Status.Status = false
	}

	if !dbusercr.Status.Status {
		instance := &kindav1beta1.DbInstance{}
		if err := r.Get(ctx, types.NamespacedName{Name: dbcr.Spec.Instance}, instance); err != nil {
			return r.manageError(ctx, dbusercr, err, false)
		}
		// Check if chosen ExtraPrivileges are allowed on the instance
		for _, priv := range dbusercr.Spec.ExtraPrivileges {
			if !slices.Contains(instance.Spec.AllowedPrivileges, priv) {
				err := fmt.Errorf("role %s is not allowed on the instance %s", priv, instance.Name)
				return r.manageError(ctx, dbusercr, err, false)
			}
		}
		db, dbuser, err := dbhelper.FetchDatabaseData(ctx, dbcr, creds, instance)
		if err != nil {
			// failed to determine database type
			return r.manageError(ctx, dbusercr, err, false)
		}

		val, ok := dbusercr.Annotations[consts.GRANT_TO_ADMIN_ON_DELETE]
		if ok {
			boolVal, err := strconv.ParseBool(val)
			if err != nil {
				log.Info(
					"can't parse a value of an annotation into a bool, ignoring",
					"annotation",
					consts.GRANT_TO_ADMIN_ON_DELETE,
					"value",
					val,
					"error",
					err,
				)
			} else {
				dbuser.GrantToAdminOnDelete = boolVal
			}
		}

		// Add extra privileges
		dbuser.ExtraPrivileges = dbusercr.Spec.ExtraPrivileges

		dbuser.GrantToAdmin = dbusercr.Spec.GrantToAdmin

		adminSecretResource, err := r.getAdminSecret(ctx, dbcr)
		if err != nil {
			// failed to get admin secret
			return r.manageError(ctx, dbusercr, err, false)
		}

		adminCred, err := db.ParseAdminCredentials(ctx, adminSecretResource.Data)
		if err != nil {
			// failed to parse database admin secret
			return r.manageError(ctx, dbusercr, err, false)
		}

		dbuser.AccessType = dbusercr.Spec.AccessType
		dbuser.Password = creds.Password
		dbuser.Username = creds.Username
		// If allow existing is set to true, db-operator will not force user creation
		// and also when a user with that property is removed, user won't be removed
		// from the database, instead only the permissions will be revoked
		existingUser := false
		// TODO: Remove this annotation
		allowExistingRaw, ok := dbusercr.Annotations[consts.ALLOW_EXISTING_USER]
		if ok {
			log.Info("Annotation is deprecated, please use .spec.existingUser instead", "annotation", consts.ALLOW_EXISTING_USER)
			existingUser, err = strconv.ParseBool(allowExistingRaw)
			if err != nil {
				log.Info(
					"can't parse a value of an annotation into a bool, ignoring",
					"annotation",
					consts.ALLOW_EXISTING_USER,
					"value",
					allowExistingRaw,
					"error",
					err,
				)
			}
		}

		if len(dbusercr.Spec.ExistingUser) > 0 {
			existingUser = true
		}

		if dbusercr.IsDeleted() {
			if commonhelper.ContainsString(dbusercr.Finalizers, "dbuser."+dbusercr.Name) {
				if err := r.handleTemplatedCredentials(ctx, dbcr, dbusercr, dbuser); err != nil {
					return r.manageError(ctx, dbusercr, err, true)
				}
				if existingUser {
					if err := database.RevokePermissions(ctx, db, dbuser, adminCred); err != nil {
						log.Error(err, "failed revoking permissions")
						return r.manageError(ctx, dbusercr, err, false)
					}
				} else {
					if err := database.DeleteUser(ctx, db, dbuser, adminCred); err != nil {
						log.Error(err, "failed deleting a user")
						return r.manageError(ctx, dbusercr, err, false)
					}
				}
				if err := r.kubeHelper.HandleDelete(ctx, userSecret); err != nil {
					return r.manageError(ctx, dbusercr, err, false)
				}
				kci.RemoveFinalizer(&dbcr.ObjectMeta, "dbuser."+dbusercr.Name)
				err = r.Update(ctx, dbcr)
				if err != nil {
					log.Error(err, "error resource updating")
					return r.manageError(ctx, dbusercr, err, false)
				}
				kci.RemoveFinalizer(&dbusercr.ObjectMeta, "dbuser."+dbusercr.Name)
				err = r.Update(ctx, dbusercr)
				if err != nil {
					log.Error(err, "error resource updating")
					return r.manageError(ctx, dbusercr, err, false)
				}
			}
		} else {
			if !dbcr.Status.Status {
				err := fmt.Errorf("database %s is not ready yet", dbcr.Name)
				return r.manageError(ctx, dbusercr, err, true)
			}

			// If allow existing is set to true, always execute UpdateOrCreate,
			// otherwise follow the old logic
			if existingUser {
				log.Info("existing user management is allowed")
				if err := database.SetPermissions(ctx, db, dbuser, adminCred); err != nil {
					return r.manageError(ctx, dbusercr, err, false)
				}
				if err = r.addFinalizers(ctx, dbusercr, dbcr); err != nil {
					return r.manageError(ctx, dbusercr, err, false)
				}
				dbusercr.Status.Created = true
			} else {
				if !dbusercr.Status.Created {
					log.Info("creating a user", "name", dbusercr.GetName())
					if err := database.CreateUser(ctx, db, dbuser, adminCred); err != nil {
						return r.manageError(ctx, dbusercr, err, false)
					}
					if err = r.addFinalizers(ctx, dbusercr, dbcr); err != nil {
						return r.manageError(ctx, dbusercr, err, false)
					}
					dbusercr.Status.Created = true
				} else {
					log.Info("updating a user", "name", dbusercr.GetName())
					if err := database.UpdateUser(ctx, db, dbuser, adminCred); err != nil {
						return r.manageError(ctx, dbusercr, err, false)
					}
				}
			}
			if err := r.handleTemplatedCredentials(ctx, dbcr, dbusercr, dbuser); err != nil {
				return r.manageError(ctx, dbusercr, err, true)
			}
			dbusercr.Status.OperatorVersion = commonhelper.OperatorVersion
			dbusercr.Status.Status = true
			dbusercr.Status.DatabaseName = dbusercr.Spec.DatabaseRef
		}
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
	annotations := dbucr.GetAnnotations()
	hash, err := kci.GenerateChecksum(dbucr.Spec)
	// just in case
	if err != nil {
		return true
	}
	return annotations["checksum/spec"] != hash ||
		annotations["checksum/secret"] != commonhelper.GenerateChecksumSecretValue(userSecret)
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
	log := log.FromContext(ctx)
	dbucr.Status.Status = false
	log.Error(issue, "an error occurred during the reconciliation")

	retryInterval := 60 * time.Second

	r.Recorder.Eventf(dbucr, nil, corev1.EventTypeWarning, "Reconcile error", "Can't reconcile", issue.Error())
	// TODO: implementing reschedule calculation based on last updated time
	return reconcile.Result{
		RequeueAfter: retryInterval,
		Requeue:      requeue,
	}, issue
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
	instance := kindav1beta1.DbInstance{}
	if err := r.Get(ctx, types.NamespacedName{Name: dbcr.Spec.Instance}, &instance); err != nil {
		return nil, err
	}

	// get database admin credentials
	secret := &corev1.Secret{}

	if err := r.Get(ctx, instance.Spec.AdminUserSecret.ToKubernetesType(), secret); err != nil {
		return nil, err
	}

	return secret, nil
}

// If dbuser has a deletion timestamp, this function will remove all the templated fields from
// secrets and configmaps, so it's a generic function that can be used for both:
// creating and removing
// It's mostly a copy-paste from the database controller, maybe it might be refactored
func (r *DbUserReconciler) handleTemplatedCredentials(ctx context.Context, dbcr *kindav1beta1.Database, dbusercr *kindav1beta1.DbUser, dbuser *database.DatabaseUser) error {
	databaseSecret, err := r.getDbUserSecret(ctx, dbusercr)
	if err != nil {
		return err
	}

	databaseConfigMap, err := r.getDatabaseConfigMap(ctx, dbcr)
	if err != nil {
		return err
	}

	creds, err := dbhelper.ParseDatabaseSecretData(dbcr, databaseSecret.Data)
	if err != nil {
		return err
	}

	// We don't need dbuser here, because if it's not nil, templates will be built for the dbuser, not the database
	instance := &kindav1beta1.DbInstance{}
	if err := r.Get(ctx, types.NamespacedName{Name: dbcr.Spec.Instance}, instance); err != nil {
		return err
	}

	db, _, err := dbhelper.FetchDatabaseData(ctx, dbcr, creds, instance)
	if err != nil {
		return err
	}

	templateds, err := templates.NewTemplateDataSource(dbcr, dbusercr, databaseSecret, databaseConfigMap, db, dbuser, instance.Spec.InstanceVars)
	if err != nil {
		return err
	}

	if !dbusercr.IsDeleted() {
		if err := templateds.Render(dbusercr.Spec.Credentials.Templates); err != nil {
			return err
		}
	} else {
		// Render with an empty slice, so templated entries are removed from Data and Annotations
		if err := templateds.Render(kindav1beta1.Templates{}); err != nil {
			return err
		}
	}

	if err := r.kubeHelper.HandleCreateOrUpdate(ctx, templateds.SecretK8sObj); err != nil {
		return err
	}

	// Set it to nil explicitly to ensure it's picked up by the GC
	templateds = nil

	return nil
}

func (r *DbUserReconciler) getDatabaseConfigMap(ctx context.Context, dbcr *kindav1beta1.Database) (*corev1.ConfigMap, error) {
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

func (r *DbUserReconciler) addFinalizers(ctx context.Context, dbusercr *kindav1beta1.DbUser, dbcr *kindav1beta1.Database) (err error) {
	log := log.FromContext(ctx)
	kci.AddFinalizer(&dbusercr.ObjectMeta, "dbuser."+dbusercr.Name)
	err = r.Update(ctx, dbusercr)
	if err != nil {
		log.Error(err, "error resource updating")
		return err
	}
	kci.AddFinalizer(&dbcr.ObjectMeta, "dbuser."+dbusercr.Name)
	err = r.Update(ctx, dbcr)
	if err != nil {
		log.Error(err, "error resource updatinsr")
		return err
	}
	return nil
}

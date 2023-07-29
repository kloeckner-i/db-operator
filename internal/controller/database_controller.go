/*
 * Copyright 2021 kloeckner.i GmbH
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
	"os"
	"strconv"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	kindav1beta1 "github.com/db-operator/db-operator/api/v1beta1"
	"github.com/db-operator/db-operator/internal/controller/backup"
	"github.com/db-operator/db-operator/pkg/config"
	"github.com/db-operator/db-operator/pkg/utils/database"
	"github.com/db-operator/db-operator/pkg/utils/kci"
	"github.com/db-operator/db-operator/pkg/utils/proxy"
	"github.com/db-operator/db-operator/pkg/utils/templates"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	crdv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	Recorder        record.EventRecorder
	Interval        time.Duration
	Conf            *config.Config
	WatchNamespaces []string
}

var (
	dbPhaseCreate               = "Creating"
	dbPhaseInstanceAccessSecret = "InstanceAccessSecretCreating"
	dbPhaseProxy                = "ProxyCreating"
	dbPhaseSecretsTemplating    = "SecretsTemplating"
	dbPhaseConfigMap            = "InfoConfigMapCreating"
	dbPhaseMonitoring           = "MonitoringCreating"
	dbPhaseBackupJob            = "BackupJobCreating"
	dbPhaseFinish               = "Finishing"
	dbPhaseReady                = "Ready"
	dbPhaseDelete               = "Deleting"
)

//+kubebuilder:rbac:groups=kinda.rocks,resources=databases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kinda.rocks,resources=databases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kinda.rocks,resources=databases/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Database object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("database", req.NamespacedName)

	reconcilePeriod := r.Interval * time.Second
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}
	// Fetch the Database custom resource
	dbcr := &kindav1beta1.Database{}
	err := r.Get(ctx, req.NamespacedName, dbcr)
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
		if err := r.Status().Update(ctx, dbcr); err != nil {
			logrus.Errorf("failed to update status - %s", err)
		}
	}()

	promDBsStatus.WithLabelValues(dbcr.Namespace, dbcr.Spec.Instance, dbcr.Name).Set(boolToFloat64(dbcr.Status.Status))
	promDBsPhase.WithLabelValues(dbcr.Namespace, dbcr.Spec.Instance, dbcr.Name).Set(dbPhaseToFloat64(dbcr.Status.Phase))

	// Check if the Database is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isDatabaseMarkedToBeDeleted := dbcr.GetDeletionTimestamp() != nil
	if isDatabaseMarkedToBeDeleted {
		dbcr.Status.Phase = dbPhaseDelete
		// Run finalization logic for database. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		if sliceContainsSubString(dbcr.ObjectMeta.Finalizers, "dbuser.") {
			err := errors.New("database can't be removed, while there are DbUser referencing it")
			logrus.Error(err)
			return r.manageError(ctx, dbcr, err, true)
		}
		if containsString(dbcr.ObjectMeta.Finalizers, "db."+dbcr.Name) {
			err := r.deleteDatabase(ctx, dbcr)
			if err != nil {
				logrus.Errorf("DB: namespace=%s, name=%s failed deleting database - %s", dbcr.Namespace, dbcr.Name, err)
				// when database deletion failed, don't requeue request. to prevent exceeding api limit (ex: against google api)
				return r.manageError(ctx, dbcr, err, false)
			}
			kci.RemoveFinalizer(&dbcr.ObjectMeta, "db."+dbcr.Name)
			err = r.Update(ctx, dbcr)
			if err != nil {
				logrus.Errorf("error resource updating - %s", err)
				return r.manageError(ctx, dbcr, err, true)
			}
		}
		// legacy finalizer just remove
		// we set owner reference for monitoring related resource instead of handling finalizer
		if containsString(dbcr.ObjectMeta.Finalizers, "monitoring."+dbcr.Name) {
			kci.RemoveFinalizer(&dbcr.ObjectMeta, "monitoring."+dbcr.Name)
			err = r.Update(ctx, dbcr)
			if err != nil {
				logrus.Errorf("error resource updating - %s", err)
				return r.manageError(ctx, dbcr, err, true)
			}
		}
		return reconcileResult, nil
	}

	databaseSecret, err := r.getDatabaseSecret(ctx, dbcr)
	if err != nil && !k8serrors.IsNotFound(err) {
		logrus.Errorf("could not get database secret - %s", err)
		return r.manageError(ctx, dbcr, err, true)
	}

	if isDBChanged(dbcr, databaseSecret) {
		logrus.Infof("DB: namespace=%s, name=%s spec changed", dbcr.Namespace, dbcr.Name)
		err := r.initialize(ctx, dbcr)
		if err != nil {
			return r.manageError(ctx, dbcr, err, true)
		}
		err = r.Status().Update(ctx, dbcr)
		if err != nil {
			logrus.Errorf("error status subresource updating - %s", err)
			return r.manageError(ctx, dbcr, err, true)
		}

		addDBChecksum(dbcr, databaseSecret)
		err = r.Update(ctx, dbcr)
		if err != nil {
			logrus.Errorf("error resource updating - %s", err)
			return r.manageError(ctx, dbcr, err, true)
		}
		logrus.Infof("DB: namespace=%s, name=%s initialized", dbcr.Namespace, dbcr.Name)
	}

	// database status not true, process phase
	if !dbcr.Status.Status {
		ownership := []metav1.OwnerReference{}
		if dbcr.Spec.Cleanup {
			ownership = append(ownership, metav1.OwnerReference{
				APIVersion: dbcr.APIVersion,
				Kind:       dbcr.Kind,
				Name:       dbcr.Name,
				UID:        dbcr.GetUID(),
			},
			)
		}

		phase := dbcr.Status.Phase
		logrus.Infof("DB: namespace=%s, name=%s start %s", dbcr.Namespace, dbcr.Name, phase)

		defer promDBsPhaseTime.WithLabelValues(phase).Observe(kci.TimeTrack(time.Now()))
		err := r.createDatabase(ctx, dbcr, ownership)
		if err != nil {
			// when database creation failed, don't requeue request. to prevent exceeding api limit (ex: against google api)
			return r.manageError(ctx, dbcr, err, false)
		}
		dbcr.Status.Phase = dbPhaseInstanceAccessSecret

		if err = r.createInstanceAccessSecret(ctx, dbcr, ownership); err != nil {
			return r.manageError(ctx, dbcr, err, true)
		}
		dbcr.Status.Phase = dbPhaseProxy
		err = r.createProxy(ctx, dbcr, ownership)
		if err != nil {
			return r.manageError(ctx, dbcr, err, true)
		}
		dbcr.Status.Phase = dbPhaseSecretsTemplating
		if err = r.createTemplatedSecrets(ctx, dbcr, ownership); err != nil {
			return r.manageError(ctx, dbcr, err, true)
		}
		dbcr.Status.Phase = dbPhaseConfigMap
		if err = r.createInfoConfigMap(ctx, dbcr, ownership); err != nil {
			return r.manageError(ctx, dbcr, err, true)
		}
		dbcr.Status.Phase = dbPhaseBackupJob
		err = r.createBackupJob(ctx, dbcr, ownership)
		if err != nil {
			return r.manageError(ctx, dbcr, err, true)
		}
		dbcr.Status.Phase = dbPhaseFinish
		dbcr.Status.Status = true
		dbcr.Status.Phase = dbPhaseReady

		err = r.Status().Update(ctx, dbcr)
		if err != nil {
			logrus.Errorf("error status subresource updating - %s", err)
			return r.manageError(ctx, dbcr, err, true)
		}
		logrus.Infof("DB: namespace=%s, name=%s finish %s", dbcr.Namespace, dbcr.Name, phase)
	}

	// status true do nothing and don't requeue
	return reconcileResult, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	eventFilter := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return isWatchedNamespace(r.WatchNamespaces, e.Object) && isDatabase(e.Object)
		}, // Reconcile only Database Create Event
		DeleteFunc: func(e event.DeleteEvent) bool {
			return isWatchedNamespace(r.WatchNamespaces, e.Object) && isDatabase(e.Object)
		}, // Reconcile only Database Delete Event
		UpdateFunc: func(e event.UpdateEvent) bool {
			return isWatchedNamespace(r.WatchNamespaces, e.ObjectNew) && isObjectUpdated(e)
		}, // Reconcile Database and Secret Update Events
		GenericFunc: func(e event.GenericEvent) bool { return true }, // Reconcile any Generic Events (operator POD or cluster restarted)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&kindav1beta1.Database{}).
		WithEventFilter(eventFilter).
		Watches(&source.Kind{Type: &corev1.Secret{}}, &secretEventHandler{r.Client}).
		Complete(r)
}

func (r *DatabaseReconciler) initialize(ctx context.Context, dbcr *kindav1beta1.Database) error {
	dbcr.Status = kindav1beta1.DatabaseStatus{}
	dbcr.Status.Status = false

	if dbcr.Spec.Instance != "" {
		instance := &kindav1beta1.DbInstance{}
		key := types.NamespacedName{
			Namespace: "",
			Name:      dbcr.Spec.Instance,
		}
		err := r.Get(ctx, key, instance)
		if err != nil {
			logrus.Errorf("DB: namespace=%s, name=%s couldn't get instance - %s", dbcr.Namespace, dbcr.Name, err)
			return err
		}

		if !instance.Status.Status {
			return errors.New("instance status not true")
		}
		dbcr.Status.InstanceRef = instance
		dbcr.Status.Phase = dbPhaseCreate
		return nil
	}
	return errors.New("instance name not defined")
}

// createDatabase secret, actual database using admin secret
func (r *DatabaseReconciler) createDatabase(ctx context.Context, dbcr *kindav1beta1.Database, ownership []metav1.OwnerReference) error {
	databaseSecret, err := r.getDatabaseSecret(ctx, dbcr)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			engine, err := dbcr.GetEngineType()
			if err != nil {
				return err
			}
			secretData, err := generateDatabaseSecretData(dbcr.ObjectMeta, engine, "")
			if err != nil {
				logrus.Errorf("can not generate credentials for database - %s", err)
				return err
			}
			newDatabaseSecret := kci.SecretBuilder(dbcr.Spec.SecretName, dbcr.Namespace, secretData, ownership)
			err = r.Create(ctx, newDatabaseSecret)
			if err != nil {
				// failed to create secret
				return err
			}
			databaseSecret = newDatabaseSecret
		} else {
			// failed to get secret resouce
			return err
		}
	}
	databaseCred, err := parseDatabaseSecretData(dbcr, databaseSecret.Data)
	if err != nil {
		// failed to parse database credential from secret
		return err
	}

	db, dbuser, err := determinDatabaseType(dbcr, databaseCred)
	if err != nil {
		// failed to determine database type
		return err
	}
	dbuser.AccessType = database.ACCESS_TYPE_MAINUSER

	adminSecretResource, err := r.getAdminSecret(ctx, dbcr)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			logrus.Errorf("can not find admin secret")
			return err
		}
		return err
	}

	// found admin secret. parse it to connect database
	adminCred, err := db.ParseAdminCredentials(adminSecretResource.Data)
	if err != nil {
		// failed to parse database admin secret
		return err
	}

	err = database.CreateDatabase(db, adminCred)
	if err != nil {
		return err
	}

	err = database.CreateOrUpdateUser(db, dbuser, adminCred)
	if err != nil {
		return err
	}

	kci.AddFinalizer(&dbcr.ObjectMeta, "db."+dbcr.Name)
	err = r.Update(ctx, dbcr)
	if err != nil {
		logrus.Errorf("error resource updating - %s", err)
		return err
	}

	err = r.annotateDatabaseSecret(ctx, dbcr, databaseSecret)
	if err != nil {
		logrus.Errorf("could not annotate database secret - %s", err)
		return err
	}

	dbcr.Status.DatabaseName = databaseCred.Name
	dbcr.Status.UserName = databaseCred.Username
	logrus.Infof("DB: namespace=%s, name=%s successfully created", dbcr.Namespace, dbcr.Name)
	return nil
}

func (r *DatabaseReconciler) deleteDatabase(ctx context.Context, dbcr *kindav1beta1.Database) error {
	if dbcr.Spec.DeletionProtected {
		logrus.Infof("DB: namespace=%s, name=%s is deletion protected. will not be deleted in backends", dbcr.Name, dbcr.Namespace)
		return nil
	}

	databaseCred := database.Credentials{
		Name:     dbcr.Status.DatabaseName,
		Username: dbcr.Status.UserName,
	}

	db, dbuser, err := determinDatabaseType(dbcr, databaseCred)
	if err != nil {
		// failed to determine database type
		return err
	}
	dbuser.AccessType = database.ACCESS_TYPE_MAINUSER

	adminSecretResource, err := r.getAdminSecret(ctx, dbcr)
	if err != nil {
		// failed to get admin secret
		return err
	}
	// found admin secret. parse it to connect database
	adminCred, err := db.ParseAdminCredentials(adminSecretResource.Data)
	if err != nil {
		// failed to parse database admin secret
		return err
	}

	err = database.DeleteDatabase(db, adminCred)
	if err != nil {
		return err
	}

	err = database.DeleteUser(db, dbuser, adminCred)
	if err != nil {
		return err
	}

	return nil
}

func (r *DatabaseReconciler) createInstanceAccessSecret(ctx context.Context, dbcr *kindav1beta1.Database, ownership []metav1.OwnerReference) error {
	if backend, _ := dbcr.GetBackendType(); backend != "google" {
		logrus.Debugf("DB: namespace=%s, name=%s %s doesn't need instance access secret skipping...", dbcr.Namespace, dbcr.Name, backend)
		return nil
	}

	var data []byte

	instance, err := dbcr.GetInstanceRef()
	if err != nil {
		return err
	}

	credFile := "credentials.json"

	if instance.Spec.Google.ClientSecret.Name != "" {
		key := instance.Spec.Google.ClientSecret.ToKubernetesType()
		secret := &corev1.Secret{}
		err := r.Get(ctx, key, secret)
		if err != nil {
			logrus.Errorf("DB: namespace=%s, name=%s can not get instance access secret", dbcr.Namespace, dbcr.Name)
			return err
		}
		data = secret.Data[credFile]
	} else {
		data, err = os.ReadFile(os.Getenv("GCSQL_CLIENT_CREDENTIALS"))
		if err != nil {
			return err
		}
	}
	secretData := make(map[string][]byte)
	secretData[credFile] = data

	newName := dbcr.InstanceAccessSecretName()
	newSecret := kci.SecretBuilder(newName, dbcr.GetNamespace(), secretData, ownership)

	err = r.Create(ctx, newSecret)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if configmap resource already exists, update
			err = r.Update(ctx, newSecret)
			if err != nil {
				logrus.Errorf("DB: namespace=%s, name=%s failed updating instance access secret", dbcr.Namespace, dbcr.Name)
				return err
			}
		} else {
			logrus.Errorf("DB: namespace=%s, name=%s failed creating instance access secret - %s", dbcr.Namespace, dbcr.Name, err)
			return err
		}
	}
	logrus.Infof("DB: namespace=%s, name=%s instance access secret created", dbcr.Namespace, dbcr.Name)
	return nil
}

func (r *DatabaseReconciler) createProxy(ctx context.Context, dbcr *kindav1beta1.Database, ownership []metav1.OwnerReference) error {
	backend, _ := dbcr.GetBackendType()
	if backend == "generic" {
		logrus.Infof("DB: namespace=%s, name=%s %s proxy creation is not yet implemented skipping...", dbcr.Namespace, dbcr.Name, backend)
		return nil
	}

	proxyInterface, err := determineProxyTypeForDB(r.Conf, dbcr)
	if err != nil {
		return err
	}

	// create proxy configmap
	cm, err := proxy.BuildConfigmap(proxyInterface, ownership)
	if err != nil {
		return err
	}
	if cm != nil { // if configmap is not null
		err = r.Create(ctx, cm)
		if err != nil {
			if k8serrors.IsAlreadyExists(err) {
				// if resource already exists, update
				err = r.Update(ctx, cm)
				if err != nil {
					logrus.Errorf("DB: namespace=%s, name=%s failed updating proxy configmap", dbcr.Namespace, dbcr.Name)
					return err
				}
			} else {
				// failed creating configmap
				logrus.Errorf("DB: namespace=%s, name=%s failed updating proxy configmap", dbcr.Namespace, dbcr.Name)
				return err
			}
		}
	}

	// create proxy deployment
	deploy, err := proxy.BuildDeployment(proxyInterface, ownership)
	if err != nil {
		return err
	}
	err = r.Create(ctx, deploy)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if resource already exists, update
			err = r.Update(ctx, deploy)
			if err != nil {
				logrus.Errorf("DB: namespace=%s, name=%s failed updating proxy deployment", dbcr.Namespace, dbcr.Name)
				return err
			}
		} else {
			// failed to create deployment
			logrus.Errorf("DB: namespace=%s, name=%s failed creating proxy deployment", dbcr.Namespace, dbcr.Name)
			return err
		}
	}

	// create proxy service
	svc, err := proxy.BuildService(proxyInterface, ownership)
	if err != nil {
		return err
	}
	err = r.Create(ctx, svc)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if resource already exists, update
			patch := client.MergeFrom(svc)
			err = r.Patch(ctx, svc, patch)
			if err != nil {
				logrus.Errorf("DB: namespace=%s, name=%s failed patching proxy service", dbcr.Namespace, dbcr.Name)
				return err
			}
		} else {
			// failed to create service
			logrus.Errorf("DB: namespace=%s, name=%s failed creating proxy service", dbcr.Namespace, dbcr.Name)
			return err
		}
	}

	crdList := crdv1.CustomResourceDefinitionList{}
	err = r.List(ctx, &crdList)
	if err != nil {
		return err
	}

	isMonitoringEnabled, err := dbcr.IsMonitoringEnabled()
	if err != nil {
		return err
	}

	if isMonitoringEnabled && inCrdList(crdList, "servicemonitors.monitoring.coreos.com") {
		// create proxy PromServiceMonitor
		promSvcMon, err := proxy.BuildServiceMonitor(proxyInterface, ownership)
		if err != nil {
			return err
		}
		err = r.Create(ctx, promSvcMon)
		if err != nil {
			if k8serrors.IsAlreadyExists(err) {
				patch := client.MergeFrom(promSvcMon)
				err := r.Patch(ctx, promSvcMon, patch)
				if err != nil {
					logrus.Errorf("DB: namespace=%s, name=%s failed patching prometheus service monitor", dbcr.Namespace, dbcr.Name)
					return err
				}
			} else {
				// failed to create service
				logrus.Errorf("DB: namespace=%s, name=%s failed creating prometehus service monitor", dbcr.Namespace, dbcr.Name)
				return err
			}
		}
	}

	engine, _ := dbcr.GetEngineType()
	dbcr.Status.ProxyStatus.ServiceName = svc.Name
	for _, svcPort := range svc.Spec.Ports {
		if svcPort.Name == engine {
			dbcr.Status.ProxyStatus.SQLPort = svcPort.Port
		}
	}
	dbcr.Status.ProxyStatus.Status = true

	logrus.Infof("DB: namespace=%s, name=%s proxy created", dbcr.Namespace, dbcr.Name)
	return nil
}

func (r *DatabaseReconciler) createTemplatedSecrets(ctx context.Context, dbcr *kindav1beta1.Database, ownership []metav1.OwnerReference) error {
	// First of all the password should be taken from secret because it's not stored anywhere else
	databaseSecret, err := r.getDatabaseSecret(ctx, dbcr)
	if err != nil {
		return err
	}

	cred, err := parseDatabaseSecretData(dbcr, databaseSecret.Data)
	if err != nil {
		return err
	}

	databaseCred, err := templates.ParseTemplatedSecretsData(dbcr, cred, databaseSecret.Data)
	if err != nil {
		return err
	}

	db, _, err := determinDatabaseType(dbcr, databaseCred)
	if err != nil {
		// failed to determine database type
		return err
	}

	dbSecrets, err := templates.GenerateTemplatedSecrets(dbcr, databaseCred, db.GetDatabaseAddress())
	if err != nil {
		return err
	}
	// Adding values
	newSecretData := templates.AppendTemplatedSecretData(dbcr, databaseSecret.Data, dbSecrets, ownership)
	newSecretData = templates.RemoveObsoleteSecret(dbcr, newSecretData, dbSecrets, ownership)

	for key, value := range newSecretData {
		databaseSecret.Data[key] = value
	}

	if err = r.Update(ctx, databaseSecret, &client.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

func (r *DatabaseReconciler) createInfoConfigMap(ctx context.Context, dbcr *kindav1beta1.Database, ownership []metav1.OwnerReference) error {
	instance, err := dbcr.GetInstanceRef()
	if err != nil {
		return err
	}

	info := instance.Status.DeepCopy().Info
	proxyStatus := dbcr.Status.ProxyStatus

	if proxyStatus.Status {
		info["DB_HOST"] = proxyStatus.ServiceName
		info["DB_PORT"] = strconv.FormatInt(int64(proxyStatus.SQLPort), 10)
	}

	sslMode, err := getSSLMode(dbcr)
	if err != nil {
		return err
	}
	info["SSL_MODE"] = sslMode
	databaseConfigResource := kci.ConfigMapBuilder(dbcr.Spec.SecretName, dbcr.Namespace, info, ownership)

	err = r.Create(ctx, databaseConfigResource)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if configmap resource already exists, update
			err = r.Update(ctx, databaseConfigResource)
			if err != nil {
				logrus.Errorf("DB: namespace=%s, name=%s failed updating database info configmap", dbcr.Namespace, dbcr.Name)
				return err
			}
		} else {
			logrus.Errorf("DB: namespace=%s, name=%s failed creating database info configmap", dbcr.Namespace, dbcr.Name)
			return err
		}
	}

	logrus.Infof("DB: namespace=%s, name=%s database info configmap created", dbcr.Namespace, dbcr.Name)
	return nil
}

func (r *DatabaseReconciler) createBackupJob(ctx context.Context, dbcr *kindav1beta1.Database, ownership []metav1.OwnerReference) error {
	if !dbcr.Spec.Backup.Enable {
		// if not enabled, skip
		return nil
	}

	cronjob, err := backup.GCSBackupCron(r.Conf, dbcr, ownership)
	if err != nil {
		return err
	}

	err = controllerutil.SetControllerReference(dbcr, cronjob, r.Scheme)
	if err != nil {
		return err
	}

	err = r.Create(ctx, cronjob)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if resource already exists, update
			err = r.Update(ctx, cronjob)
			if err != nil {
				logrus.Errorf("DB: namespace=%s, name=%s failed updating backup cronjob", dbcr.Namespace, dbcr.Name)
				return err
			}
		} else {
			// failed to create deployment
			logrus.Errorf("DB: namespace=%s, name=%s failed creating backup cronjob", dbcr.Namespace, dbcr.Name)
			return err
		}
	}

	return nil
}

func (r *DatabaseReconciler) getDatabaseSecret(ctx context.Context, dbcr *kindav1beta1.Database) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	key := types.NamespacedName{
		Namespace: dbcr.Namespace,
		Name:      dbcr.Spec.SecretName,
	}
	err := r.Get(ctx, key, secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

func (r *DatabaseReconciler) annotateDatabaseSecret(ctx context.Context, dbcr *kindav1beta1.Database, secret *corev1.Secret) error {
	annotations := secret.ObjectMeta.GetAnnotations()
	if len(annotations) == 0 {
		annotations = make(map[string]string)
	}
	annotations[DbSecretAnnotation] = dbcr.Name
	secret.ObjectMeta.SetAnnotations(annotations)

	return r.Update(ctx, secret)
}

func (r *DatabaseReconciler) getAdminSecret(ctx context.Context, dbcr *kindav1beta1.Database) (*corev1.Secret, error) {
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

func (r *DatabaseReconciler) manageError(ctx context.Context, dbcr *kindav1beta1.Database, issue error, requeue bool) (reconcile.Result, error) {
	dbcr.Status.Status = false
	logrus.Errorf("DB: namespace=%s, name=%s failed %s - %s", dbcr.Namespace, dbcr.Name, dbcr.Status.Phase, issue)
	promDBsPhaseError.WithLabelValues(dbcr.Status.Phase).Inc()

	retryInterval := 60 * time.Second

	r.Recorder.Event(dbcr, "Warning", "Failed"+dbcr.Status.Phase, issue.Error())
	err := r.Status().Update(ctx, dbcr)
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

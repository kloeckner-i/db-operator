/*
 * Copyright 2021 kloeckner.i GmbH
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
	"maps"
	"os"
	"strconv"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kindav1beta1 "github.com/db-operator/db-operator/v2/api/v1beta1"
	"github.com/db-operator/db-operator/v2/internal/controller/backup"
	commonhelper "github.com/db-operator/db-operator/v2/internal/helpers/common"
	dbhelper "github.com/db-operator/db-operator/v2/internal/helpers/database"
	kubehelper "github.com/db-operator/db-operator/v2/internal/helpers/kube"
	proxyhelper "github.com/db-operator/db-operator/v2/internal/helpers/proxy"
	"github.com/db-operator/db-operator/v2/internal/utils/templates"
	"github.com/db-operator/db-operator/v2/pkg/config"
	"github.com/db-operator/db-operator/v2/pkg/consts"
	"github.com/db-operator/db-operator/v2/pkg/utils/database"
	"github.com/db-operator/db-operator/v2/pkg/utils/kci"
	"github.com/db-operator/db-operator/v2/pkg/utils/proxy"
	secTemplates "github.com/db-operator/db-operator/v2/pkg/utils/templates"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	crdv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
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
	Recorder        events.EventRecorder
	Interval        time.Duration
	Conf            *config.Config
	WatchNamespaces []string
	CheckChanges    bool
	kubeHelper      *kubehelper.KubeHelper
}

var (
	dbPhaseReconcile            = "Reconciling"
	dbPhaseCreateOrUpdate       = "CreatingOrUpdating"
	dbPhaseInstanceAccessSecret = "InstanceAccessSecretCreating"
	dbPhaseProxy                = "ProxyCreating"
	dbPhaseSecretsTemplating    = "SecretsTemplating"
	dbPhaseConfigMap            = "InfoConfigMapCreating"
	dbPhaseTemplating           = "Templating"
	dbPhaseBackupJob            = "BackupJobCreating"
	dbPhaseReady                = "Ready"
	dbPhaseDelete               = "Deleting"
)

//+kubebuilder:rbac:groups=kinda.rocks,resources=databases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kinda.rocks,resources=databases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kinda.rocks,resources=databases/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets;configmaps,verbs=get;list;watch;create;update
//+kubebuilder:rbac:groups="events.k8s.io",resources=events,verbs=create
//+kubebuilder:rbac:groups="batch",resources=cronjob,verbs=create;update;get;watch

func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	phase := dbPhaseReconcile
	log := log.FromContext(ctx)
	reconcilePeriod := r.Interval
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
			log.Error(err, "failed to update status")
		}
	}()

	promDBsStatus.WithLabelValues(dbcr.Namespace, dbcr.Spec.Instance, dbcr.Name).Set(boolToFloat64(dbcr.Status.Status))

	// Init the kubehelper object
	r.kubeHelper = kubehelper.NewKubeHelper(r.Client, r.Recorder, dbcr)
	/* ----------------------------------------------------------------
	 * -- Check if the Database is marked to be deleted, which is
	 * --  indicated by the deletion timestamp being set.
	 * ------------------------------------------------------------- */
	if dbcr.IsDeleted() {
		return r.handleDbDelete(ctx, dbcr)
	}

	/* ----------------------------------------------------------------
	 * -- If db can't be accessed, set the status to false.
	 * -- It doesn't make sense to run healthCheck if InstanceRef
	 * --  is not set, because it means that database initialization
	 * --  wasn't triggered.
	 * ------------------------------------------------------------- */
	if dbcr.Status.Engine != "" {
		if err := r.healthCheck(ctx, dbcr); err != nil {
			log.Info("Healthcheck is failed")
			dbcr.Status.Status = false
		}
	}

	/* ----------------------------------------------------------------
	 * -- Check if db-operator must fire sql actions, or just
	 * --  update the k8s resources
	 * ------------------------------------------------------------- */
	mustReconile, err := r.isFullReconcile(ctx, dbcr)
	if err != nil {
		return r.manageError(ctx, dbcr, err, true, phase)
	}

	return r.handleDbCreateOrUpdate(ctx, dbcr, mustReconile)
}

func (r *DatabaseReconciler) healthCheck(ctx context.Context, dbcr *kindav1beta1.Database) error {
	log := log.FromContext(ctx)
	if len(dbcr.Spec.ExistingUser) > 0 {
		log.Info("An existing user is used, running health check as admin")
		return r.healthCheckAdmin(ctx, dbcr)
	}
	return r.healthCheckUser(ctx, dbcr)
}

// Move it to helpers and start testing it
func (r *DatabaseReconciler) healthCheckUser(ctx context.Context, dbcr *kindav1beta1.Database) error {
	var dbSecret *corev1.Secret
	dbSecret, err := r.getDatabaseSecret(ctx, dbcr)
	if err != nil {
		return err
	}

	databaseCred, err := dbhelper.ParseDatabaseSecretData(dbcr, dbSecret.Data)
	if err != nil {
		// failed to parse database credential from secret
		return err
	}

	instance := &kindav1beta1.DbInstance{}
	if err := r.Get(ctx, types.NamespacedName{Name: dbcr.Spec.Instance}, instance); err != nil {
		return err
	}

	db, dbuser, err := dbhelper.FetchDatabaseData(ctx, dbcr, databaseCred, instance)
	if err != nil {
		// failed to determine database type
		return err
	}

	if err := db.CheckStatus(ctx, dbuser); err != nil {
		return err
	}

	return nil
}

// Move it to helpers and start testing it
func (r *DatabaseReconciler) healthCheckAdmin(ctx context.Context, dbcr *kindav1beta1.Database) error {
	var dbSecret *corev1.Secret
	dbSecret, err := r.getDatabaseSecret(ctx, dbcr)
	if err != nil {
		return err
	}

	databaseCred, err := dbhelper.ParseDatabaseSecretData(dbcr, dbSecret.Data)
	if err != nil {
		// failed to parse database credential from secret
		return err
	}

	instance := &kindav1beta1.DbInstance{}
	if err := r.Get(ctx, types.NamespacedName{Name: dbcr.Spec.Instance}, instance); err != nil {
		return err
	}

	db, _, err := dbhelper.FetchDatabaseData(ctx, dbcr, databaseCred, instance)
	if err != nil {
		// failed to determine database type
		return err
	}

	adminSecretResource, err := r.getAdminSecret(ctx, dbcr)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return err
		}
		return err
	}

	// found admin secret. parse it to connect database
	adminCred, err := db.ParseAdminCredentials(ctx, adminSecretResource.Data)
	if err != nil {
		// failed to parse database admin secret
		return err
	}

	if err := db.CheckStatus(ctx, adminCred); err != nil {
		return err
	}

	return nil
}

/* --------------------------------------------------------------------
 * -- isFullReconcile returns true if db-operator should fire database
 * --  queries. If user doesn't want to load database server with
 * --  with db-operator queries, there is an option to execute them
 * --  only when db-operator detects the change between the wished
 * --  config and the existing one, or the special annotation is set.
 * -- Otherwise, if user doesn't care about the load, since it
 * --  actually should be very low, the checkForChanges flag can
 * --  be set to false, and then db-operator will run queries
 * --  against the database.
 * ----------------------------------------------------------------- */
func (r *DatabaseReconciler) isFullReconcile(ctx context.Context, dbcr *kindav1beta1.Database) (bool, error) {
	log := log.FromContext(ctx)
	// This is the first check, because even if the checkForChanges is false,
	// the annotation is expected to be removed
	if _, ok := dbcr.GetAnnotations()[consts.DATABASE_FORCE_FULL_RECONCILE]; ok {
		r.Recorder.Eventf(dbcr, nil, corev1.EventTypeNormal, "Reconciling", "Force full reconcile", "Annotations %s was found", consts.DATABASE_FORCE_FULL_RECONCILE)

		annotations := dbcr.GetAnnotations()
		delete(annotations, consts.DATABASE_FORCE_FULL_RECONCILE)
		err := r.Update(ctx, dbcr)
		if err != nil {
			log.Error(err, "error resource updating")
			return false, err
		}
		return true, nil
	}

	// If we don't check changes, then just always reconcile
	if !r.CheckChanges {
		return true, nil
	}

	// If database is not healthy, reconcile
	if !dbcr.Status.Status {
		return true, nil
	}

	var dbSecret *corev1.Secret
	var err error

	dbSecret, err = r.getDatabaseSecret(ctx, dbcr)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return true, nil
		} else {
			return false, err
		}
	}

	return commonhelper.IsDBChanged(dbcr, dbSecret), nil
}

/* --------------------------------------------------------------------
 * -- This function should be called when a database is created
 * --  or updated.
 * -- The mustReconcile argument is the trigger for the database
 * --  action to run. If mustReconcile is true, all the db queries
 * --  will be executed.
 * ------------------------------------------------------------------ */
func (r *DatabaseReconciler) handleDbCreateOrUpdate(ctx context.Context, dbcr *kindav1beta1.Database, mustReconcile bool) (reconcile.Result, error) {
	log := log.FromContext(ctx)
	var err error

	phase := dbPhaseCreateOrUpdate

	reconcilePeriod := r.Interval * time.Second
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	log.Info("reconciling database")

	defer promDBsPhaseTime.WithLabelValues(phase).Observe(kci.TimeTrack(time.Now()))

	// Handle the secret creation
	var dbSecret *corev1.Secret
	dbSecret, err = r.getDatabaseSecret(ctx, dbcr)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			if err := r.setEngine(ctx, dbcr); err != nil {
				return r.manageError(ctx, dbcr, err, true, phase)
			}
			dbSecret, err = r.createSecret(ctx, dbcr)
			if err != nil {
				return r.manageError(ctx, dbcr, err, false, phase)
			}
		} else {
			return r.manageError(ctx, dbcr, err, false, phase)
		}
	}

	// Apply extra metadata from the Database credentials spec to the
	// credentials Secret before it is created or updated in the cluster.
	// This ensures that changes to .spec.credentials.metadata are
	// reconciled onto the Secret on subsequent reconciliations.
	if dbcr.Spec.Credentials.Metadata != nil {
		meta := dbcr.Spec.Credentials.Metadata
		if len(meta.ExtraLabels) > 0 {
			if dbSecret.Labels == nil {
				dbSecret.Labels = map[string]string{}
			}
			for k, v := range meta.ExtraLabels {
				dbSecret.Labels[k] = v
			}
		}
		if len(meta.ExtraAnnotations) > 0 {
			if dbSecret.Annotations == nil {
				dbSecret.Annotations = map[string]string{}
			}
			for k, v := range meta.ExtraAnnotations {
				dbSecret.Annotations[k] = v
			}
		}
	}

	if err := r.kubeHelper.ModifyObject(ctx, dbSecret); err != nil {
		return r.manageError(ctx, dbcr, err, true, phase)
	}

	// Create
	if mustReconcile {
		if err := r.setEngine(ctx, dbcr); err != nil {
			return r.manageError(ctx, dbcr, err, false, phase)
		}

		if err := r.createDatabase(ctx, dbcr, dbSecret); err != nil {
			// when database creation failed, don't requeue request. to prevent exceeding api limit (ex: against google api)
			return r.manageError(ctx, dbcr, err, false, phase)
		}
	}

	phase = dbPhaseInstanceAccessSecret
	if err := r.handleInstanceAccessSecret(ctx, dbcr); err != nil {
		return r.manageError(ctx, dbcr, err, true, phase)
	}
	log.Info("instance access secret created")

	phase = dbPhaseProxy
	err = r.handleProxy(ctx, dbcr)
	if err != nil {
		return r.manageError(ctx, dbcr, err, true, phase)
	}

	phase = dbPhaseSecretsTemplating
	if err = r.createTemplatedSecrets(ctx, dbcr); err != nil {
		return r.manageError(ctx, dbcr, err, true, phase)
	}
	phase = dbPhaseConfigMap
	if err = r.handleInfoConfigMap(ctx, dbcr); err != nil {
		return r.manageError(ctx, dbcr, err, true, phase)
	}
	phase = dbPhaseTemplating

	// A temporary check that exists to avoid creating templates if secretsTemplates are used.
	// todo: It should be removed when secretsTemlates are gone

	if len(dbcr.Spec.SecretsTemplates) == 0 {
		if err := r.handleTemplatedCredentials(ctx, dbcr); err != nil {
			return r.manageError(ctx, dbcr, err, false, phase)
		}
	}
	phase = dbPhaseBackupJob
	err = r.handleBackupJob(ctx, dbcr)
	if err != nil {
		return r.manageError(ctx, dbcr, err, true, phase)
	}

	dbcr.Status.Status = true
	phase = dbPhaseReady

	err = r.Status().Update(ctx, dbcr)
	if err != nil {
		log.Error(err, "error status subresource updating")
		return r.manageError(ctx, dbcr, err, true, phase)
	}

	return reconcileResult, nil
}

func (r *DatabaseReconciler) handleDbDelete(ctx context.Context, dbcr *kindav1beta1.Database) (reconcile.Result, error) {
	log := log.FromContext(ctx)
	phase := dbPhaseDelete
	reconcilePeriod := r.Interval * time.Second
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	// Run finalization logic for database. If the
	// finalization logic fails, don't remove the finalizer so
	// that we can retry during the next reconciliation.
	if commonhelper.SliceContainsSubString(dbcr.Finalizers, "dbuser.") {
		err := errors.New("database can't be removed, while there are DbUser referencing it")
		return r.manageError(ctx, dbcr, err, true, phase)
	}

	if commonhelper.ContainsString(dbcr.Finalizers, "db."+dbcr.Name) {
		err := r.deleteDatabase(ctx, dbcr)
		if err != nil {
			log.Error(err, "failed deleting database")
			// when database deletion failed, don't requeue request. to prevent exceeding api limit (ex: against google api)
			return r.manageError(ctx, dbcr, err, false, phase)
		}
		kci.RemoveFinalizer(&dbcr.ObjectMeta, "db."+dbcr.Name)
		err = r.Update(ctx, dbcr)
		if err != nil {
			log.Error(err, "error resource updating")
			return r.manageError(ctx, dbcr, err, true, phase)
		}
	}
	// A temporary check that exists to avoid creating templates if secretsTemplates are used.
	// todo: It should be removed when secretsTemlates are gone
	if len(dbcr.Spec.SecretsTemplates) == 0 {
		if err := r.handleTemplatedCredentials(ctx, dbcr); err != nil {
			return r.manageError(ctx, dbcr, err, false, phase)
		}
	}

	if err := r.handleInstanceAccessSecret(ctx, dbcr); err != nil {
		return r.manageError(ctx, dbcr, err, true, phase)
	}

	if err := r.handleProxy(ctx, dbcr); err != nil {
		return r.manageError(ctx, dbcr, err, true, phase)
	}

	if err := r.handleInfoConfigMap(ctx, dbcr); err != nil {
		return r.manageError(ctx, dbcr, err, true, phase)
	}

	if err := r.handleBackupJob(ctx, dbcr); err != nil {
		return r.manageError(ctx, dbcr, err, true, phase)
	}

	dbSecret, err := r.getDatabaseSecret(ctx, dbcr)

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return r.manageError(ctx, dbcr, err, true, phase)
		}
	} else {
		if err := r.kubeHelper.ModifyObject(ctx, dbSecret); err != nil {
			return r.manageError(ctx, dbcr, err, true, phase)
		}
	}

	return reconcileResult, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kindav1beta1.Database{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findDatabaseForSecret),
		).
		Complete(r)
}

func (r *DatabaseReconciler) findDatabaseForSecret(ctx context.Context, secret client.Object) []reconcile.Request {
	log := log.FromContext(ctx)
	name := secret.GetName()
	namespace := secret.GetNamespace()
	labels := secret.GetLabels()
	log = log.WithValues("secret", name, "namespace", namespace)
	log.Info("A secret modification was spotted")
	kind, ok := labels[consts.USED_BY_KIND_LABEL_KEY]
	if !ok {
		log.Info("Secret is not used by anything")
		return nil
	}
	if kind != "Database" {
		log.Info("Kind is not database")
		return nil
	}
	databaseName, ok := labels[consts.USED_BY_NAME_LABEL_KEY]
	if !ok {
		log.Info("Name is not found on the secret")
		return nil
	}
	log.Info("Getting databases for the secret")
	database := &kindav1beta1.Database{}

	if err := r.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      databaseName,
	}, database); err != nil {
		log.Error(err, "Couldn't get the database", "database", databaseName)
		return nil
	}

	log.Info("Got databases", "database", databaseName)
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      databaseName,
			Namespace: namespace,
		},
	}
	return []reconcile.Request{request}
}

func (r *DatabaseReconciler) setEngine(ctx context.Context, dbcr *kindav1beta1.Database) error {
	log := log.FromContext(ctx)
	if len(dbcr.Spec.Instance) == 0 {
		return errors.New("instance name not defined")
	}

	if len(dbcr.Status.Engine) == 0 {
		instance := &kindav1beta1.DbInstance{}
		key := types.NamespacedName{
			Namespace: "",
			Name:      dbcr.Spec.Instance,
		}
		err := r.Get(ctx, key, instance)
		if err != nil {
			log.Error(err, "couldn't get instance")
			return err
		}

		if !instance.Status.Status {
			return errors.New("instance status not true")
		}

		dbcr.Status.Engine = instance.Spec.Engine
		if err := r.Status().Update(ctx, dbcr); err != nil {
			return err
		}
		return nil
	}

	return nil
}

// createDatabase secret, actual database using admin secret
func (r *DatabaseReconciler) createDatabase(ctx context.Context, dbcr *kindav1beta1.Database, dbSecret *corev1.Secret) error {
	log := log.FromContext(ctx)
	databaseCred, err := dbhelper.ParseDatabaseSecretData(dbcr, dbSecret.Data)
	if err != nil {
		// failed to parse database credential from secret
		return err
	}
	instance := &kindav1beta1.DbInstance{}
	if err := r.Get(ctx, types.NamespacedName{Name: dbcr.Spec.Instance}, instance); err != nil {
		return err
	}

	db, dbuser, err := dbhelper.FetchDatabaseData(ctx, dbcr, databaseCred, instance)
	if err != nil {
		// failed to determine database type
		return err
	}
	dbuser.AccessType = database.ACCESS_TYPE_MAINUSER

	adminSecretResource, err := r.getAdminSecret(ctx, dbcr)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Error(err, "can not find admin secret")
			return err
		}
		return err
	}

	// found admin secret. parse it to connect database
	adminCred, err := db.ParseAdminCredentials(ctx, adminSecretResource.Data)
	if err != nil {
		// failed to parse database admin secret
		return err
	}

	err = database.CreateDatabase(ctx, db, adminCred)
	if err != nil {
		return err
	}
	if len(dbcr.Spec.ExistingUser) > 0 {
		if err := database.SetPermissions(ctx, db, dbuser, adminCred); err != nil {
			return err
		}
	} else {
		if err := database.CreateOrUpdateUser(ctx, db, dbuser, adminCred); err != nil {
			return err
		}
	}
	if len(dbcr.Status.ExtraGrants) > 0 && !instance.Spec.AllowExtraGrants {
		err := errors.New("extra grants are not allowed on the instance")
		return err
	}
	if instance.Spec.AllowExtraGrants {
		for _, existingExtraGrant := range dbcr.Status.ExtraGrants {
			if !existingExtraGrant.IsExtraGrant(dbcr.Spec.ExtraGrants) {
				log.Info("Removing an extra grant from the db", "username", existingExtraGrant.User, "accessType", existingExtraGrant.AccessType)
				// Creating a dbuser instance without a password,
				// only to use it for revoking permissions
				userGrant := &database.DatabaseUser{
					Username:             existingExtraGrant.User,
					AccessType:           existingExtraGrant.AccessType,
					GrantToAdmin:         dbuser.GrantToAdmin,
					GrantToAdminOnDelete: dbuser.GrantToAdminOnDelete,
				}
				if err := database.RevokePermissions(ctx, db, userGrant, adminCred); err != nil {
					return err
				}
			}
		}

		for _, extraGrant := range dbcr.Spec.ExtraGrants {
			if !extraGrant.IsExtraGrant(dbcr.Status.ExtraGrants) {
				log.Info("Adding an extra grant to the db", "username", extraGrant.User, "accessType", extraGrant.AccessType)
				// Creating a dbuser instance without a password,
				// only to use it for setting permissions
				userGrant := &database.DatabaseUser{
					Username:             extraGrant.User,
					AccessType:           extraGrant.AccessType,
					GrantToAdmin:         dbuser.GrantToAdmin,
					GrantToAdminOnDelete: dbuser.GrantToAdminOnDelete,
				}
				if err := database.SetPermissions(ctx, db, userGrant, adminCred); err != nil {
					return err
				}

			}
		}
	}

	dbcr.Status.ExtraGrants = dbcr.Spec.ExtraGrants
	if err := r.Status().Update(ctx, dbcr); err != nil {
		return err
	}
	// if only in status.extraGrant revoke permissions
	kci.AddFinalizer(&dbcr.ObjectMeta, "db."+dbcr.Name)

	commonhelper.AddDBChecksum(dbcr, dbSecret)
	err = r.Update(ctx, dbcr)
	if err != nil {
		return err
	}

	err = r.Update(ctx, dbcr)
	if err != nil {
		log.Error(err, "error resource updating")
		return err
	}

	dbcr.Status.OperatorVersion = commonhelper.OperatorVersion
	dbcr.Status.DatabaseName = databaseCred.Name
	dbcr.Status.UserName = databaseCred.Username
	log.Info("successfully created")
	return nil
}

func (r *DatabaseReconciler) deleteDatabase(ctx context.Context, dbcr *kindav1beta1.Database) error {
	log := log.FromContext(ctx)
	if dbcr.Spec.DeletionProtected {
		log.Info("database is deletion protected, it will not be deleted in backends")
		return nil
	}

	databaseCred := database.Credentials{
		Name:     dbcr.Status.DatabaseName,
		Username: dbcr.Status.UserName,
	}

	instance := &kindav1beta1.DbInstance{}
	if err := r.Get(ctx, types.NamespacedName{Name: dbcr.Spec.Instance}, instance); err != nil {
		return err
	}

	db, dbuser, err := dbhelper.FetchDatabaseData(ctx, dbcr, databaseCred, instance)
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
	adminCred, err := db.ParseAdminCredentials(ctx, adminSecretResource.Data)
	if err != nil {
		// failed to parse database admin secret
		return err
	}

	err = database.DeleteDatabase(ctx, db, adminCred)
	if err != nil {
		return err
	}

	// We can't revoke permissions on a removed database
	if len(dbcr.Spec.ExistingUser) == 0 {
		if err := database.DeleteUser(ctx, db, dbuser, adminCred); err != nil {
			return err
		}
	}

	return nil
}

func (r *DatabaseReconciler) handleInstanceAccessSecret(ctx context.Context, dbcr *kindav1beta1.Database) error {
	log := log.FromContext(ctx)
	var err error
	instance := &kindav1beta1.DbInstance{}
	if err := r.Get(ctx, types.NamespacedName{Name: dbcr.Spec.Instance}, instance); err != nil {
		return err
	}

	if backend, _ := instance.GetBackendType(); backend != "google" {
		log.V(2).Info("doesn't need instance access secret skipping...")
		return nil
	}

	var data []byte

	credFile := "credentials.json"

	if instance.Spec.Google.ClientSecret.Name != "" {
		key := instance.Spec.Google.ClientSecret.ToKubernetesType()
		secret := &corev1.Secret{}
		err := r.Get(ctx, key, secret)
		if err != nil {
			log.Error(err, "can not get instance access secret")
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
	newSecret := kci.SecretBuilder(newName, dbcr.GetNamespace(), secretData)
	if err := r.kubeHelper.ModifyObject(ctx, newSecret); err != nil {
		return err
	}

	return nil
}

func (r *DatabaseReconciler) handleProxy(ctx context.Context, dbcr *kindav1beta1.Database) error {
	log := log.FromContext(ctx)
	instance := &kindav1beta1.DbInstance{}
	if err := r.Get(ctx, types.NamespacedName{Name: dbcr.Spec.Instance}, instance); err != nil {
		return err
	}

	backend, _ := instance.GetBackendType()
	if backend == "generic" {
		log.Info("proxy creation is not yet implemented skipping...")
		return nil
	}

	proxyInterface, err := proxyhelper.DetermineProxyTypeForDB(ctx, r.Conf, dbcr, instance)
	if err != nil {
		return err
	}

	// create proxy configmap
	cm, err := proxy.BuildConfigmap(ctx, proxyInterface)
	if err != nil {
		return err
	}
	if cm != nil {
		if err := r.kubeHelper.ModifyObject(ctx, cm); err != nil {
			return err
		}
	}

	// create proxy deployment
	deploy, err := proxy.BuildDeployment(ctx, proxyInterface)
	if err != nil {
		return err
	}
	if err := r.kubeHelper.ModifyObject(ctx, deploy); err != nil {
		return err
	}

	// create proxy service
	svc, err := proxy.BuildService(ctx, proxyInterface)
	if err != nil {
		return err
	}
	if err := r.kubeHelper.ModifyObject(ctx, svc); err != nil {
		return err
	}

	crdList := crdv1.CustomResourceDefinitionList{}
	err = r.List(ctx, &crdList)
	if err != nil {
		return err
	}

	isMonitoringEnabled := instance.IsMonitoringEnabled()
	if isMonitoringEnabled && commonhelper.InCrdList(crdList, "servicemonitors.monitoring.coreos.com") {
		// create proxy PromServiceMonitor
		promSvcMon, err := proxy.BuildServiceMonitor(ctx, proxyInterface)
		if err != nil {
			return err
		}
		if err := r.kubeHelper.ModifyObject(ctx, promSvcMon); err != nil {
			return err
		}

	}

	dbcr.Status.ProxyStatus.ServiceName = svc.Name
	for _, svcPort := range svc.Spec.Ports {
		if svcPort.Name == dbcr.Status.Engine {
			dbcr.Status.ProxyStatus.SQLPort = svcPort.Port
		}
	}
	dbcr.Status.ProxyStatus.Status = true

	log.Info("proxy is created")
	return nil
}

// If database has a deletion timestamp, this function will remove all the templated fields from
// secrets and configmaps, so it's a generic function that can be used for both:
// creating and removing
func (r *DatabaseReconciler) handleTemplatedCredentials(ctx context.Context, dbcr *kindav1beta1.Database) error {
	databaseSecret, err := r.getDatabaseSecret(ctx, dbcr)
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

	db, dbuser, err := dbhelper.FetchDatabaseData(ctx, dbcr, creds, instance)
	if err != nil {
		return err
	}

	templateds, err := templates.NewTemplateDataSource(dbcr, nil, databaseSecret, databaseConfigMap, db, dbuser, instance.Spec.InstanceVars)
	if err != nil {
		return err
	}

	if !dbcr.IsDeleted() {
		if err := templateds.Render(dbcr.Spec.Credentials.Templates); err != nil {
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

	if err := r.kubeHelper.HandleCreateOrUpdate(ctx, templateds.ConfigMapK8sObj); err != nil {
		return err
	}
	// Set it to nil explicitly to ensure it's picked up by the GC
	templateds = nil
	return nil
}

func (r *DatabaseReconciler) createTemplatedSecrets(ctx context.Context, dbcr *kindav1beta1.Database) error {
	if len(dbcr.Spec.SecretsTemplates) > 0 {
		r.Recorder.Eventf(dbcr, nil, corev1.EventTypeWarning, "Deprecation", "Secrets Templates",
			"secretsTemplates are deprecated and will be removed in the next API version. Please consider using templates",
		)
		// First of all the password should be taken from secret because it's not stored anywhere else
		databaseSecret, err := r.getDatabaseSecret(ctx, dbcr)
		if err != nil {
			return err
		}

		cred, err := dbhelper.ParseDatabaseSecretData(dbcr, databaseSecret.Data)
		if err != nil {
			return err
		}

		databaseCred, err := secTemplates.ParseTemplatedSecretsData(ctx, dbcr, cred, databaseSecret.Data)
		if err != nil {
			return err
		}
		instance := &kindav1beta1.DbInstance{}
		if err := r.Get(ctx, types.NamespacedName{Name: dbcr.Spec.Instance}, instance); err != nil {
			return err
		}

		db, _, err := dbhelper.FetchDatabaseData(ctx, dbcr, databaseCred, instance)
		if err != nil {
			// failed to determine database type
			return err
		}
		dbSecrets, err := secTemplates.GenerateTemplatedSecrets(ctx, dbcr, databaseCred, db.GetDatabaseAddress(ctx))
		if err != nil {
			return err
		}
		// Adding values
		newSecretData := secTemplates.AppendTemplatedSecretData(ctx, dbcr, databaseSecret.Data, dbSecrets)
		newSecretData = secTemplates.RemoveObsoleteSecret(ctx, dbcr, newSecretData, dbSecrets)

		maps.Copy(databaseSecret.Data, newSecretData)

		if err := r.kubeHelper.ModifyObject(ctx, databaseSecret); err != nil {
			return err
		}

		if err = r.Update(ctx, databaseSecret, &client.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (r *DatabaseReconciler) handleInfoConfigMap(ctx context.Context, dbcr *kindav1beta1.Database) error {
	log := log.FromContext(ctx)
	instance := &kindav1beta1.DbInstance{}
	if err := r.Get(ctx, types.NamespacedName{Name: dbcr.Spec.Instance}, instance); err != nil {
		return err
	}

	info := instance.Status.DeepCopy().Info
	proxyStatus := dbcr.Status.ProxyStatus

	if proxyStatus.Status {
		info["DB_HOST"] = proxyStatus.ServiceName
		info["DB_PORT"] = strconv.FormatInt(int64(proxyStatus.SQLPort), 10)
	}

	sslMode, err := dbhelper.GetSSLMode(dbcr, instance)
	if err != nil {
		return err
	}
	info["SSL_MODE"] = sslMode
	databaseConfigResource := kci.ConfigMapBuilder(dbcr.Spec.SecretName, dbcr.Namespace, info)

	if err := r.kubeHelper.ModifyObject(ctx, databaseConfigResource); err != nil {
		return err
	}

	log.Info("database info configmap created")
	return nil
}

func (r *DatabaseReconciler) handleBackupJob(ctx context.Context, dbcr *kindav1beta1.Database) error {
	if !dbcr.Spec.Backup.Enable {
		// if not enabled, skip
		return nil
	}
	instance := &kindav1beta1.DbInstance{}
	if err := r.Get(ctx, types.NamespacedName{Name: dbcr.Spec.Instance}, instance); err != nil {
		return err
	}

	cronjob, err := backup.BackupCronJobManifest(r.Conf, dbcr, instance)
	if err != nil {
		return err
	}

	err = controllerutil.SetControllerReference(dbcr, cronjob, r.Scheme)
	if err != nil {
		return err
	}

	if err := r.kubeHelper.ModifyObject(ctx, cronjob); err != nil {
		return err
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

func (r *DatabaseReconciler) getDatabaseConfigMap(ctx context.Context, dbcr *kindav1beta1.Database) (*corev1.ConfigMap, error) {
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

func (r *DatabaseReconciler) getAdminSecret(ctx context.Context, dbcr *kindav1beta1.Database) (*corev1.Secret, error) {
	instance := &kindav1beta1.DbInstance{}
	if err := r.Get(ctx, types.NamespacedName{Name: dbcr.Spec.Instance}, instance); err != nil {
		return nil, err
	}

	// get database admin credentials
	secret := &corev1.Secret{}

	if err := r.Get(ctx, instance.Spec.AdminUserSecret.ToKubernetesType(), secret); err != nil {
		return nil, err
	}

	return secret, nil
}

func (r *DatabaseReconciler) manageError(ctx context.Context, dbcr *kindav1beta1.Database, issue error, requeue bool, phase string) (reconcile.Result, error) {
	dbcr.Status.Status = false
	log := log.FromContext(ctx)
	log.Error(issue, "an error occurred during the reconciliation")
	promDBsPhaseError.WithLabelValues(phase).Inc()

	retryInterval := 60 * time.Second

	r.Recorder.Eventf(dbcr, nil, corev1.EventTypeWarning, "Reconcile error", "Can't reconcile", issue.Error())
	err := r.Status().Update(ctx, dbcr)
	if err != nil {
		log.Error(err, "unable to update status")
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

func (r *DatabaseReconciler) createSecret(ctx context.Context, dbcr *kindav1beta1.Database) (*corev1.Secret, error) {
	log := log.FromContext(ctx)
	secretData, err := dbhelper.GenerateDatabaseSecretData(dbcr.ObjectMeta, dbcr.Status.Engine, "", dbcr.Spec.ExistingUser)
	if err != nil {
		log.Error(err, "can not generate credentials for database")
		return nil, err
	}

	databaseSecret := kci.SecretBuilder(dbcr.Spec.SecretName, dbcr.Namespace, secretData)
	return databaseSecret, nil
}

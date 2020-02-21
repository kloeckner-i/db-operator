package database

import (
	"context"
	"errors"
	"time"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	config "github.com/kloeckner-i/db-operator/pkg/config"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// var log = logf.Log.WithName("controller_database")
var conf = config.Config{}

// Add creates a new Database Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDatabase{client: mgr.GetClient(), scheme: mgr.GetScheme(), recorder: mgr.GetEventRecorderFor("database-controller")}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("database-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Database
	err = c.Watch(&source.Kind{Type: &kciv1alpha1.Database{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Database
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kciv1alpha1.Database{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileDatabase implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileDatabase{}

// ReconcileDatabase reconciles a Database object
type ReconcileDatabase struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

var (
	phaseCreate               = "Creating"
	phaseConfigMap            = "InfoConfigMapCreating"
	phaseInstanceAccessSecret = "InstanceAccessSecretCreating"
	phaseProxy                = "ProxyCreating"
	phaseMonitoring           = "MonitoringCreating"
	phaseBackupJob            = "BackupJobCreating"
	phaseFinish               = "Finishing"
	phaseReady                = "Ready"
	phaseDelete               = "Deleting"
)

// GCSQLClientSecretName used as secret name containing service account json key with Cloud SQL Client role
var GCSQLClientSecretName = "cloudsql-instance-credentials"

// Reconcile reads that state of the cluster for a Database object and makes changes based on the state read
// and what is in the Database.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileDatabase) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	// reqLogger.Info("Reconciling Database")

	reconcilePeriod := 60 * time.Second
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	ctx := context.TODO()

	// Fetch the Database custom resource
	dbcr := &kciv1alpha1.Database{}
	err := r.client.Get(ctx, request.NamespacedName, dbcr)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcileResult, nil
		}
		// Error reading the object - requeue the request.
		return reconcileResult, err
	}

	// Update object status always when function exit abnormally or through a panic.
	defer r.client.Status().Update(context.Background(), dbcr)

	promDBsStatus.WithLabelValues(dbcr.Namespace, dbcr.Spec.Instance, dbcr.Name).Set(boolToFloat64(dbcr.Status.Status))
	promDBsPhase.WithLabelValues(dbcr.Namespace, dbcr.Spec.Instance, dbcr.Name).Set(dbPhaseToFloat64(dbcr.Status.Phase))

	// Check if the Database is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isDatabaseMarkedToBeDeleted := dbcr.GetDeletionTimestamp() != nil
	if isDatabaseMarkedToBeDeleted {
		dbcr.Status.Phase = phaseDelete
		// Run finalization logic for database. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		if containsString(dbcr.ObjectMeta.Finalizers, "db."+dbcr.Name) {
			logrus.Infof("DB: namespace=%s, name=%s deleting database", dbcr.Namespace, dbcr.Name)
			err := r.deleteDatabase(dbcr)
			if err != nil {
				logrus.Errorf("DB: namespace=%s, name=%s failed deleting database - %s", dbcr.Namespace, dbcr.Name, err)
				// when database deletion failed, don't requeue request. to prevent exceeding api limit (ex: against google api)
				return r.manageError(dbcr, err, false)
			}
			kci.RemoveFinalizer(&dbcr.ObjectMeta, "db."+dbcr.Name)
			err = r.client.Update(context.Background(), dbcr)
			if err != nil {
				logrus.Errorf("error resource updating - %s", err)
				return r.manageError(dbcr, err, true)
			}
		}
		// legacy finalizer just remove
		// we set owner reference for monitoring related resource instead of handling finalizer
		if containsString(dbcr.ObjectMeta.Finalizers, "monitoring."+dbcr.Name) {
			kci.RemoveFinalizer(&dbcr.ObjectMeta, "monitoring."+dbcr.Name)
			err = r.client.Update(context.Background(), dbcr)
			if err != nil {
				logrus.Errorf("error resource updating - %s", err)
				return r.manageError(dbcr, err, true)
			}
		}
		return reconcileResult, nil
	}

	if isSpecChanged(dbcr) {
		logrus.Infof("DB: namespace=%s, name=%s spec changed", dbcr.Namespace, dbcr.Name)
		err := r.initialize(dbcr)
		if err != nil {
			return r.manageError(dbcr, err, true)
		}
		err = r.client.Status().Update(context.Background(), dbcr)
		if err != nil {
			logrus.Errorf("error status subresource updating - %s", err)
			return r.manageError(dbcr, err, true)
		}

		addSpecChecksum(dbcr)
		err = r.client.Update(context.Background(), dbcr)
		if err != nil {
			logrus.Errorf("error resource updating - %s", err)
			return r.manageError(dbcr, err, true)
		}
		logrus.Infof("DB: namespace=%s, name=%s initialized", dbcr.Namespace, dbcr.Name)
		return reconcileResult, nil
	}

	// database status not true, process phase
	if !dbcr.Status.Status {
		phase := dbcr.Status.Phase
		logrus.Infof("DB: namespace=%s, name=%s start %s", dbcr.Namespace, dbcr.Name, phase)

		defer promDBsPhaseTime.WithLabelValues(phase).Observe(kci.TimeTrack(time.Now()))

		switch phase {
		case phaseCreate:
			err := r.createDatabase(dbcr)
			if err != nil {
				logrus.Errorf("DB: namespace=%s, name=%s failed creating database - %s", dbcr.Namespace, dbcr.Name, err)
				// when database creation failed, don't requeue request. to prevent exceeding api limit (ex: against google api)
				return r.manageError(dbcr, err, false)
			}
			kci.AddFinalizer(&dbcr.ObjectMeta, "db."+dbcr.Name)
			err = r.client.Update(context.Background(), dbcr)
			if err != nil {
				logrus.Errorf("error resource updating - %s", err)
				return r.manageError(dbcr, err, true)
			}
			dbcr.Status.Phase = phaseInstanceAccessSecret
		case phaseInstanceAccessSecret:
			err := r.createInstanceAccessSecret(dbcr)
			if err != nil {
				return r.manageError(dbcr, err, true)
			}
			dbcr.Status.Phase = phaseProxy
		case phaseProxy:
			err := r.createProxy(dbcr)
			if err != nil {
				return r.manageError(dbcr, err, true)
			}
			dbcr.Status.Phase = phaseBackupJob
		case phaseBackupJob:
			err := r.createBackupJob(dbcr)
			if err != nil {
				return r.manageError(dbcr, err, true)
			}
			dbcr.Status.Phase = phaseMonitoring
		case phaseMonitoring:
			err := r.createMonitoringExporter(dbcr)
			if err != nil {
				return r.manageError(dbcr, err, true)
			}
			dbcr.Status.Phase = phaseFinish
		case phaseFinish:
			dbcr.Status.Status = true
			dbcr.Status.Phase = phaseReady
		case phaseReady:
			return reconcileResult, nil //do nothing and don't requeue
		default:
			logrus.Errorf("DB: namespace=%s, name=%s unknown phase %s", dbcr.Namespace, dbcr.Name, phase)
			err := r.initialize(dbcr)
			if err != nil {
				return r.manageError(dbcr, err, true)
			} // set phase to initial state
			return r.manageError(dbcr, errors.New("unknown phase"), false)
		}

		err = r.client.Status().Update(context.Background(), dbcr)
		if err != nil {
			logrus.Errorf("error status subresource updating - %s", err)
			return r.manageError(dbcr, err, true)
		}

		logrus.Infof("DB: namespace=%s, name=%s finish %s", dbcr.Namespace, dbcr.Name, phase)
		return reconcileResult, nil // success phase
	}

	// status true do nothing and don't requeue
	return reconcileResult, nil
}

func (r *ReconcileDatabase) initialize(dbcr *kciv1alpha1.Database) error {
	dbcr.Status = kciv1alpha1.DatabaseStatus{}
	dbcr.Status.Status = false

	if dbcr.Spec.Instance != "" {
		instance := &kciv1alpha1.DbInstance{}
		key := types.NamespacedName{
			Namespace: "",
			Name:      dbcr.Spec.Instance,
		}
		err := r.client.Get(context.TODO(), key, instance)
		if err != nil {
			logrus.Errorf("DB: namespace=%s, name=%s couldn't get instance - %s", dbcr.Namespace, dbcr.Name, err)
			return err
		}

		if !instance.Status.Status {
			return errors.New("instance status not true")
		}
		dbcr.Status.InstanceRef = instance
		dbcr.Status.Phase = phaseCreate
		return nil
	}
	return errors.New("instance name not defined")
}

func (r *ReconcileDatabase) manageError(dbcr *kciv1alpha1.Database, issue error, requeue bool) (reconcile.Result, error) {
	dbcr.Status.Status = false
	logrus.Errorf("DB: namespace=%s, name=%s failed %s - %s", dbcr.Namespace, dbcr.Name, dbcr.Status.Phase, issue)
	promDBsPhaseError.WithLabelValues(dbcr.Status.Phase).Inc()

	var retryInterval = 60 * time.Second

	r.recorder.Event(dbcr, "Warning", "Failed"+dbcr.Status.Phase, issue.Error())
	err := r.client.Status().Update(context.Background(), dbcr)
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

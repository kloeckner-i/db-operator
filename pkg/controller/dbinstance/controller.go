package dbinstance

import (
	"context"
	"errors"
	"time"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"

	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// var log = logf.Log.WithName("controller_dbinstance")

// Add creates a new DbInstance Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDbInstance{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("dbinstance-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource DbInstance
	err = c.Watch(&source.Kind{Type: &kciv1alpha1.DbInstance{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileDbInstance implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileDbInstance{}

// ReconcileDbInstance reconciles a DbInstance object
type ReconcileDbInstance struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

var (
	phaseValidate    = "Valitating"
	phaseCreate      = "Creating"
	phaseBroadcast   = "Broadcasting"
	phaseProxyCreate = "ProxyCreating"
	phaseRunning     = "Running"
)

// Reconcile reads that state of the cluster for a DbInstance object and makes changes based on the state read
// and what is in the DbInstance.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileDbInstance) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	// reqLogger.Info("Reconciling DbInstance")
	reconcilePeriod := 60 * time.Second
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	ctx := context.TODO()

	// Fetch the DbInstance custom resource
	dbin := &kciv1alpha1.DbInstance{}
	err := r.client.Get(ctx, request.NamespacedName, dbin)
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

	// Update object status always when function returns, either normally or through a panic.
	defer r.client.Status().Update(context.Background(), dbin)

	// Check if spec changed
	if isChanged(dbin) {
		logrus.Infof("Instance: name=%s spec changed", dbin.Name)
		dbin.Status.Status = false
		dbin.Status.Phase = phaseValidate // set phase to initial state
	}

	phase := dbin.Status.Phase
	logrus.Infof("Instance: name=%s %s", dbin.Name, dbin.Status.Phase)
	defer promDBInstancesPhaseTime.WithLabelValues(phase).Observe(time.Since(time.Now()).Seconds())
	promDBInstancesPhase.WithLabelValues(dbin.Name).Set(dbInstancePhaseToFloat64(phase))

	switch phase {
	case phaseValidate:
		dbin.Status.Status = false
		if err := dbin.ValidateBackend(); err != nil {
			return reconcileResult, err
		}

		if err := dbin.ValidateEngine(); err != nil {
			return reconcileResult, err
		}

		addChecksumStatus(dbin)
		dbin.Status.Phase = phaseCreate
		dbin.Status.Info = map[string]string{}
	case phaseCreate:
		err := r.create(dbin)
		if err != nil {
			logrus.Errorf("Instance: name=%s instance creation failed - %s", dbin.Name, err)
			return reconcileResult, nil // failed but don't requeue the request. retry by changing spec or config
		}
		dbin.Status.Status = true
		dbin.Status.Phase = phaseBroadcast
	case phaseBroadcast:
		err := r.broadcast(dbin)
		if err != nil {
			logrus.Errorf("Instance: name=%s broadcasting failed - %s", dbin.Name, err)
			return reconcileResult, err
		}
		dbin.Status.Phase = phaseProxyCreate
	case phaseProxyCreate:
		err := r.createProxy(dbin)
		if err != nil {
			logrus.Errorf("Instance: name=%s proxy creation failed - %s", dbin.Name, err)
			return reconcileResult, err
		}
		dbin.Status.Phase = phaseRunning
	case phaseRunning:
		return reconcileResult, nil //do nothing and don't requeue
	default:
		logrus.Errorf("Instance: name=%s unknown phase %s", dbin.Name, phase)
		dbin.Status.Phase = phaseValidate // set phase to initial state
		return reconcileResult, errors.New("unknown phase")
	}

	// dbinstance created successfully - don't requeue
	return reconcileResult, nil
}

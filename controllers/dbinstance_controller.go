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
	"time"

	"github.com/go-logr/logr"
	kciv1alpha1 "github.com/kloeckner-i/db-operator/api/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/config"
	"github.com/kloeckner-i/db-operator/pkg/utils/database"
	"github.com/kloeckner-i/db-operator/pkg/utils/dbinstance"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	"github.com/kloeckner-i/db-operator/pkg/utils/proxy"
	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	dbInstancePhaseValidate    = "Valitating"
	dbInstancePhaseCreate      = "Creating"
	dbInstancePhaseBroadcast   = "Broadcasting"
	dbInstancePhaseProxyCreate = "ProxyCreating"
	dbInstancePhaseRunning     = "Running"
)

// DbInstanceReconciler reconciles a DbInstance object
type DbInstanceReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Interval time.Duration
	Conf     *config.Config
}

//+kubebuilder:rbac:groups=kci.rocks,resources=dbinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kci.rocks,resources=dbinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kci.rocks,resources=dbinstances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DbInstance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *DbInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("dbinstance", req.NamespacedName)

	reconcilePeriod := r.Interval * time.Second
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	// Fetch the DbInstance custom resource
	dbin := &kciv1alpha1.DbInstance{}
	err := r.Get(ctx, req.NamespacedName, dbin)
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
	defer func() {
		if err := r.Status().Update(ctx, dbin); err != nil {
			logrus.Errorf("failed to update status - %s", err)
		}
	}()

	// Check if spec changed
	if isDBInstanceSpecChanged(ctx, dbin) {
		logrus.Infof("Instance: name=%s spec changed", dbin.Name)
		dbin.Status.Status = false
		dbin.Status.Phase = dbInstancePhaseValidate // set phase to initial state
	}

	phase := dbin.Status.Phase
	logrus.Infof("Instance: name=%s %s", dbin.Name, phase)
	defer promDBInstancesPhaseTime.WithLabelValues(phase).Observe(time.Since(time.Now()).Seconds())
	promDBInstancesPhase.WithLabelValues(dbin.Name).Set(dbInstancePhaseToFloat64(phase))

	switch phase {
	case dbInstancePhaseValidate:
		dbin.Status.Status = false
		if err := dbin.ValidateBackend(); err != nil {
			return reconcileResult, err
		}

		if err := dbin.ValidateEngine(); err != nil {
			return reconcileResult, err
		}

		addDBInstanceChecksumStatus(ctx, dbin)
		dbin.Status.Phase = dbInstancePhaseCreate
		dbin.Status.Info = map[string]string{}
	case dbInstancePhaseCreate:
		err := r.create(ctx, dbin)
		if err != nil {
			logrus.Errorf("Instance: name=%s instance creation failed - %s", dbin.Name, err)
			return reconcileResult, nil // failed but don't requeue the request. retry by changing spec or config
		}
		dbin.Status.Status = true
		dbin.Status.Phase = dbInstancePhaseBroadcast
	case dbInstancePhaseBroadcast:
		err := r.broadcast(ctx, dbin)
		if err != nil {
			logrus.Errorf("Instance: name=%s broadcasting failed - %s", dbin.Name, err)
			return reconcileResult, err
		}
		dbin.Status.Phase = dbInstancePhaseProxyCreate
	case dbInstancePhaseProxyCreate:
		err := r.createProxy(ctx, dbin)
		if err != nil {
			logrus.Errorf("Instance: name=%s proxy creation failed - %s", dbin.Name, err)
			return reconcileResult, err
		}
		dbin.Status.Phase = dbInstancePhaseRunning
	case dbInstancePhaseRunning:
		return reconcileResult, nil // do nothing and don't requeue
	default:
		logrus.Errorf("Instance: name=%s unknown phase %s", dbin.Name, phase)
		dbin.Status.Phase = dbInstancePhaseValidate // set phase to initial state
		return reconcileResult, errors.New("unknown phase")
	}

	// dbinstance created successfully - don't requeue
	return reconcileResult, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DbInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kciv1alpha1.DbInstance{}).
		Complete(r)
}

func (r *DbInstanceReconciler) create(ctx context.Context, dbin *kciv1alpha1.DbInstance) error {
	secret, err := kci.GetSecretResource(ctx, dbin.Spec.AdminUserSecret.ToKubernetesType())
	if err != nil {
		logrus.Errorf("Instance: name=%s failed to get instance admin user secret %s/%s", dbin.Name, dbin.Spec.AdminUserSecret.Namespace, dbin.Spec.AdminUserSecret.Name)
		return err
	}

	db := database.New(dbin.Spec.Engine)
	cred, err := db.ParseAdminCredentials(secret.Data)
	if err != nil {
		return err
	}

	backend, err := dbin.GetBackendType()
	if err != nil {
		return err
	}

	var instance dbinstance.DbInstance
	switch backend {
	case "google":
		configmap, err := kci.GetConfigResource(ctx, dbin.Spec.Google.ConfigmapName.ToKubernetesType())
		if err != nil {
			logrus.Errorf("Instance: name=%s reading GCSQL instance config %s/%s", dbin.Name, dbin.Spec.Google.ConfigmapName.Namespace, dbin.Spec.Google.ConfigmapName.Name)
			return err
		}

		name := dbin.Spec.Google.InstanceName
		config := configmap.Data["config"]
		user := cred.Username
		password := cred.Password
		apiEndpoint := dbin.Spec.Google.APIEndpoint

		instance = dbinstance.GsqlNew(name, config, user, password, apiEndpoint)
	case "generic":
		instance = &dbinstance.Generic{
			Host:         dbin.Spec.Generic.Host,
			Port:         dbin.Spec.Generic.Port,
			PublicIP:     dbin.Spec.Generic.PublicIP,
			Engine:       dbin.Spec.Engine,
			User:         cred.Username,
			Password:     cred.Password,
			SSLEnabled:   dbin.Spec.SSLConnection.Enabled,
			SkipCAVerify: dbin.Spec.SSLConnection.SkipVerify,
		}
	default:
		return errors.New("not supported backend type")
	}

	info, err := dbinstance.Create(instance)
	if err != nil {
		if err == dbinstance.ErrAlreadyExists {
			logrus.Debugf("Instance: name=%s instance already exists in backend, updating instance", dbin.Name)
			info, err = dbinstance.Update(instance)
			if err != nil {
				logrus.Errorf("Instance: name=%s failed updating instance - %s", dbin.Name, err)
				return err
			}
		} else {
			logrus.Errorf("Instance: name=%s failed creating instance - %s", dbin.Name, err)
			return err
		}
	}

	dbin.Status.Info = info
	return nil
}

func (r *DbInstanceReconciler) broadcast(ctx context.Context, dbin *kciv1alpha1.DbInstance) error {
	dbList := &kciv1alpha1.DatabaseList{}
	err := r.List(ctx, dbList)
	if err != nil {
		return err
	}

	for _, db := range dbList.Items {
		if db.Spec.Instance == dbin.Name {
			annotations := db.ObjectMeta.GetAnnotations()
			if _, found := annotations["checksum/spec"]; found {
				annotations["checksum/spec"] = ""
				db.ObjectMeta.SetAnnotations(annotations)
				err = r.Update(ctx, &db)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (r *DbInstanceReconciler) createProxy(ctx context.Context, dbin *kciv1alpha1.DbInstance) error {
	proxyInterface, err := determineProxyTypeForInstance(r.Conf, dbin)
	if err != nil {
		if err == ErrNoProxySupport {
			return nil
		}
		return err
	}

	// create proxy deployment
	deploy, err := proxy.BuildDeployment(proxyInterface)
	if err != nil {
		return err
	}
	err = r.Create(ctx, deploy)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if resource already exists, update
			err = r.Update(ctx, deploy)
			if err != nil {
				logrus.Errorf("Instance: name=%s failed updating proxy deployment", dbin.Name)
				return err
			}
		} else {
			// failed to create deployment
			logrus.Errorf("Instance: name=%s failed creating proxy deployment", dbin.Name)
			return err
		}
	}

	// create proxy service
	svc, err := proxy.BuildService(proxyInterface)
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
				logrus.Errorf("Instance: name=%s failed patching proxy service", dbin.Name)
				return err
			}
		} else {
			// failed to create service
			logrus.Errorf("Instance: name=%s failed creating proxy service", dbin.Name)
			return err
		}
	}

	return nil
}

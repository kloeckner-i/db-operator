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

package controller

import (
	"context"
	"errors"
	"strconv"
	"time"

	kindav1beta1 "github.com/db-operator/db-operator/v2/api/v1beta1"
	commonhelper "github.com/db-operator/db-operator/v2/internal/helpers/common"
	kubehelper "github.com/db-operator/db-operator/v2/internal/helpers/kube"
	proxyhelper "github.com/db-operator/db-operator/v2/internal/helpers/proxy"
	"github.com/db-operator/db-operator/v2/pkg/config"
	"github.com/db-operator/db-operator/v2/pkg/consts"
	"github.com/db-operator/db-operator/v2/pkg/utils/database"
	"github.com/db-operator/db-operator/v2/pkg/utils/dbinstance"
	"github.com/db-operator/db-operator/v2/pkg/utils/proxy"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	dbInstancePhaseValidate    = "Validating"
	dbInstancePhaseCreate      = "Creating"
	dbInstancePhaseBroadcast   = "Broadcasting"
	dbInstancePhaseProxyCreate = "ProxyCreating"
	dbInstancePhaseRunning     = "Running"
)

// DbInstanceReconciler reconciles a DbInstance object
type DbInstanceReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	Interval   time.Duration
	Recorder   events.EventRecorder
	Conf       *config.Config
	kubeHelper *kubehelper.KubeHelper
}

//+kubebuilder:rbac:groups=kinda.rocks,resources=dbinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kinda.rocks,resources=dbinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kinda.rocks,resources=dbinstances/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets;configmaps,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="events.k8s.io",resources=events,verbs=get;list;watch;update;patch;create

func (r *DbInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	reconcilePeriod := r.Interval
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	// Fetch the DbInstance custom resource
	dbin := &kindav1beta1.DbInstance{}
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
			log.Error(err, "failed to update status")
		}
	}()

	r.kubeHelper = kubehelper.NewKubeHelper(r.Client, r.Recorder, dbin)

	// Fetch data for checksum and reconcile
	instanceData, err := r.fetchInstanceData(ctx, dbin)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Error(err, "referenced secret/configmap not found")
			return reconcileResult, nil // Wait for resource to be created, or for user to fix spec
		}
		return reconcileResult, err
	}

	// Label referenced resources
	if err := r.labelReferencedResources(ctx, dbin, instanceData); err != nil {
		log.Error(err, "failed to label referenced resources")
		return reconcileResult, err
	}

	// Check if spec or referenced data changed
	if commonhelper.IsDBInstanceChanged(ctx, dbin, instanceData) {
		log.Info("spec or referenced data changed")
		dbin.Status.Status = false
		dbin.Status.Phase = dbInstancePhaseValidate // set phase to initial state
	}

	phase := dbin.Status.Phase

	start := time.Now()
	defer func() { promDBInstancesPhaseTime.WithLabelValues(phase).Observe(time.Since(start).Seconds()) }()

	promDBInstancesPhase.WithLabelValues(dbin.Name).Set(dbInstancePhaseToFloat64(phase))
	if !dbin.Status.Status {
		if err := dbin.ValidateBackend(); err != nil {
			return reconcileResult, err
		}

		if err := dbin.ValidateEngine(); err != nil {
			return reconcileResult, err
		}

		dbin.Status.Checksums = commonhelper.GenerateDBInstanceChecksums(dbin, instanceData)
		dbin.Status.Phase = dbInstancePhaseCreate
		dbin.Status.Info = map[string]string{}

		err = r.create(ctx, dbin, instanceData)
		if err != nil {
			log.Error(err, "instance creation failed")
			return reconcileResult, nil // failed but don't requeue the request. retry by changing spec or config
		}
		dbin.Status.Status = true
		dbin.Status.Phase = dbInstancePhaseBroadcast

		err = r.broadcast(ctx, dbin)
		if err != nil {
			log.Error(err, "broadcasting failed")
			return reconcileResult, err
		}
		dbin.Status.Phase = dbInstancePhaseProxyCreate

		err = r.createProxy(ctx, dbin, []metav1.OwnerReference{})
		if err != nil {
			log.Error(err, "proxy creation failed")
			return reconcileResult, err
		}
		dbin.Status.Phase = dbInstancePhaseRunning
	}

	return reconcileResult, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DbInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kindav1beta1.DbInstance{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findDbInstanceForResource),
		).
		Watches(
			&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(r.findDbInstanceForResource),
		).
		Complete(r)
}

func (r *DbInstanceReconciler) findDbInstanceForResource(ctx context.Context, obj client.Object) []reconcile.Request {
	labels := obj.GetLabels()
	if dbInstanceName, ok := labels[consts.DBINSTANCE_NAME_LABEL_KEY]; ok {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name: dbInstanceName,
				},
			},
		}
	}
	return nil
}

func (r *DbInstanceReconciler) fetchInstanceData(ctx context.Context, dbin *kindav1beta1.DbInstance) (commonhelper.DbInstanceData, error) {
	data := commonhelper.DbInstanceData{}

	// Fetch Admin Secret
	adminSecret := &corev1.Secret{}
	err := r.Get(ctx, dbin.Spec.AdminUserSecret.ToKubernetesType(), adminSecret)
	if err != nil {
		return data, err
	}
	data.AdminSecret = adminSecret

	if backend, _ := dbin.GetBackendType(); backend == "google" {
		configMap := &corev1.ConfigMap{}
		err := r.Get(ctx, dbin.Spec.Google.ConfigmapName.ToKubernetesType(), configMap)
		if err != nil {
			return data, err
		}
		data.ConfigMap = configMap

		if dbin.Spec.Google.ClientSecret.Name != "" {
			clientSecret := &corev1.Secret{}
			err := r.Get(ctx, dbin.Spec.Google.ClientSecret.ToKubernetesType(), clientSecret)
			if err != nil {
				return data, err
			}
			data.ClientSecret = clientSecret
		}
	}

	if backend, _ := dbin.GetBackendType(); backend == "generic" {
		if from := dbin.Spec.Generic.HostFrom; from != nil {
			obj, err := r.fetchFromRef(ctx, from)
			if err != nil {
				return data, err
			}
			data.HostFrom = obj
		}
		if from := dbin.Spec.Generic.PortFrom; from != nil {
			obj, err := r.fetchFromRef(ctx, from)
			if err != nil {
				return data, err
			}
			data.PortFrom = obj
		}
		if from := dbin.Spec.Generic.PublicIPFrom; from != nil {
			obj, err := r.fetchFromRef(ctx, from)
			if err != nil {
				return data, err
			}
			data.PublicIPFrom = obj
		}
	}

	return data, nil
}

func (r *DbInstanceReconciler) fetchFromRef(ctx context.Context, from *kindav1beta1.FromRef) (client.Object, error) {
	var obj client.Object
	switch from.Kind {
	case "Secret":
		obj = &corev1.Secret{}
	case "ConfigMap":
		obj = &corev1.ConfigMap{}
	default:
		return nil, errors.New("invalid kind in fromRef")
	}

	err := r.Get(ctx, from.ToKubernetesType(), obj)
	return obj, err
}

func (r *DbInstanceReconciler) create(ctx context.Context, dbin *kindav1beta1.DbInstance, data commonhelper.DbInstanceData) error {
	log := log.FromContext(ctx)

	db := database.New(dbin.Spec.Engine)
	cred, err := db.ParseAdminCredentials(ctx, data.AdminSecret.Data)
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
		name := dbin.Spec.Google.InstanceName
		config := data.ConfigMap.Data["config"]
		user := cred.Username
		password := cred.Password
		apiEndpoint := dbin.Spec.Google.APIEndpoint

		instance = dbinstance.GsqlNew(ctx, name, config, user, password, apiEndpoint)
	case "generic":
		var host string
		var port uint16
		var publicIP string

		if from := dbin.Spec.Generic.HostFrom; from != nil {
			host, err = r.kubeHelper.GetValueFrom(ctx, from.Kind, from.Namespace, from.Name, from.Key)
			if err != nil {
				return err
			}
		} else {
			host = dbin.Spec.Generic.Host
		}

		if from := dbin.Spec.Generic.PortFrom; from != nil {
			portStr, err := r.kubeHelper.GetValueFrom(ctx, from.Kind, from.Namespace, from.Name, from.Key)
			if err != nil {
				return err
			}
			port64, err := strconv.ParseUint(portStr, 10, 16)
			if err != nil {
				return err
			}
			port = uint16(port64)
		} else {
			port = dbin.Spec.Generic.Port
		}

		if from := dbin.Spec.Generic.PublicIPFrom; from != nil {
			publicIP, err = r.kubeHelper.GetValueFrom(ctx, from.Kind, from.Namespace, from.Name, from.Key)
			if err != nil {
				return err
			}
		} else {
			publicIP = dbin.Spec.Generic.PublicIP
		}
		instance = &dbinstance.Generic{
			Host:         host,
			Port:         port,
			PublicIP:     publicIP,
			Engine:       dbin.Spec.Engine,
			User:         cred.Username,
			Password:     cred.Password,
			SSLEnabled:   dbin.Spec.SSLConnection.Enabled,
			SkipCAVerify: dbin.Spec.SSLConnection.SkipVerify,
		}
	default:
		return errors.New("not supported backend type")
	}

	info, err := dbinstance.Create(ctx, instance)
	if err != nil {
		if err == dbinstance.ErrAlreadyExists {
			log.V(2).Info("instance already exists in backend, updating instance")
			info, err = dbinstance.Update(ctx, instance)
			if err != nil {
				log.Error(err, "failed updating instance")
				return err
			}
		} else {
			log.Error(err, "failed creating instance")
			return err
		}
	}

	dbin.Status.Info = info
	return nil
}

func (r *DbInstanceReconciler) broadcast(ctx context.Context, dbin *kindav1beta1.DbInstance) error {
	dbList := &kindav1beta1.DatabaseList{}
	err := r.List(ctx, dbList)
	if err != nil {
		return err
	}

	for _, db := range dbList.Items {
		if db.Spec.Instance == dbin.Name {
			annotations := db.GetAnnotations()
			if _, found := annotations["checksum/spec"]; found {
				annotations["checksum/spec"] = ""
				db.SetAnnotations(annotations)
				err = r.Update(ctx, &db)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (r *DbInstanceReconciler) createProxy(ctx context.Context, dbin *kindav1beta1.DbInstance, _ []metav1.OwnerReference) error {
	log := log.FromContext(ctx)
	proxyInterface, err := proxyhelper.DetermineProxyTypeForInstance(ctx, r.Conf, dbin)
	if err != nil {
		if err == proxyhelper.ErrNoProxySupport {
			return nil
		}
		return err
	}

	// Create proxy deployment
	deploy, err := proxy.BuildDeployment(ctx, proxyInterface)
	if err != nil {
		log.Error(err, "failed to build proxy deployment")
		return err
	}
	err = r.Create(ctx, deploy)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if resource already exists, update
			err = r.Update(ctx, deploy)
			if err != nil {
				log.Error(err, "failed to update proxy deployment")
				return err
			}
		} else {
			// failed to create deployment
			log.Error(err, "failed to create proxy deployment")
			return err
		}
	}

	// Create proxy service
	svc, err := proxy.BuildService(ctx, proxyInterface)
	if err != nil {
		log.Error(err, "failed to build proxy service")
		return err
	}
	err = r.Create(ctx, svc)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if resource already exists, update
			patch := client.MergeFrom(svc)
			err = r.Patch(ctx, svc, patch)
			if err != nil {
				log.Error(err, "failed to patch proxy service")
				return err
			}
		} else {
			// failed to create service
			log.Error(err, "failed to create proxy service")
			return err
		}
	}

	return nil
}

func (r *DbInstanceReconciler) labelReferencedResources(ctx context.Context, dbin *kindav1beta1.DbInstance, data commonhelper.DbInstanceData) error {
	if data.AdminSecret != nil {
		if err := commonhelper.EnsureLabel(ctx, r.Client, data.AdminSecret, consts.DBINSTANCE_NAME_LABEL_KEY, dbin.Name); err != nil {
			return err
		}
	}

	if data.ConfigMap != nil {
		if err := commonhelper.EnsureLabel(ctx, r.Client, data.ConfigMap, consts.DBINSTANCE_NAME_LABEL_KEY, dbin.Name); err != nil {
			return err
		}
	}

	if data.ClientSecret != nil {
		if err := commonhelper.EnsureLabel(ctx, r.Client, data.ClientSecret, consts.DBINSTANCE_NAME_LABEL_KEY, dbin.Name); err != nil {
			return err
		}
	}

	if data.HostFrom != nil {
		if err := commonhelper.EnsureLabel(ctx, r.Client, data.HostFrom, consts.DBINSTANCE_NAME_LABEL_KEY, dbin.Name); err != nil {
			return err
		}
	}

	if data.PortFrom != nil {
		if err := commonhelper.EnsureLabel(ctx, r.Client, data.PortFrom, consts.DBINSTANCE_NAME_LABEL_KEY, dbin.Name); err != nil {
			return err
		}
	}

	if data.PublicIPFrom != nil {
		if err := commonhelper.EnsureLabel(ctx, r.Client, data.PublicIPFrom, consts.DBINSTANCE_NAME_LABEL_KEY, dbin.Name); err != nil {
			return err
		}
	}

	return nil
}

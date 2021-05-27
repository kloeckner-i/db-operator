package database

import (
	"context"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	proxy "github.com/kloeckner-i/db-operator/pkg/utils/proxy"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileDatabase) createProxy(dbcr *kciv1alpha1.Database) error {
	backend, _ := dbcr.GetBackendType()
	if backend == "generic" {
		logrus.Infof("DB: namespace=%s, name=%s %s proxy creation is not yet implemented skipping...", dbcr.Namespace, dbcr.Name, backend)
		return nil
	}

	if backend == "percona" {
		err := r.replicateMonitorUserSecret(dbcr)
		if err != nil {
			return err
		}
	}

	proxyInterface, err := determinProxyType(r.conf, dbcr)
	if err != nil {
		return err
	}

	// create proxy configmap
	cm, err := proxy.BuildConfigmap(proxyInterface)
	if err != nil {
		return err
	}
	if cm != nil { // if configmap is not null
		err = r.client.Create(context.TODO(), cm)
		if err != nil {
			if k8serrors.IsAlreadyExists(err) {
				// if resource already exists, update
				err = r.client.Update(context.TODO(), cm)
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
	deploy, err := proxy.BuildDeployment(proxyInterface)
	if err != nil {
		return err
	}
	err = r.client.Create(context.TODO(), deploy)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if resource already exists, update
			err = r.client.Update(context.TODO(), deploy)
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
	svc, err := proxy.BuildService(proxyInterface)
	if err != nil {
		return err
	}
	err = r.client.Create(context.TODO(), svc)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if resource already exists, update
			patch := client.MergeFrom(svc)
			err = r.client.Patch(context.TODO(), svc, patch)
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

func (r *ReconcileDatabase) replicateMonitorUserSecret(dbcr *kciv1alpha1.Database) error {
	dbin, err := dbcr.GetInstanceRef()
	if err != nil {
		return err
	}
	source := dbin.Spec.DbInstanceSource

	key := source.Percona.MonitorUserSecret
	monitorUserSecret := &v1.Secret{}

	err = r.client.Get(context.TODO(), key, monitorUserSecret)
	if err != nil {
		logrus.Errorf("DB: namespace=%s, name=%s couldn't get monitor user secret - %s", dbcr.Namespace, dbcr.Name, err)
		return err
	}

	newSecret := kci.SecretBuilder(dbcr.Name+"-proxysql-monitoruser", dbcr.Namespace, monitorUserSecret.Data)
	err = r.client.Create(context.TODO(), newSecret)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if resource already exists, update
			err = r.client.Update(context.TODO(), monitorUserSecret)
			if err != nil {
				logrus.Errorf("DB: namespace=%s, name=%s failed replicating monitor user secret", dbcr.Namespace, dbcr.Name)
				return err
			}
		} else {
			// failed to create deployment
			logrus.Errorf("DB: namespace=%s, name=%s failed replicating monitor user secret", dbcr.Namespace, dbcr.Name)
			return err
		}
	}

	dbcr.Status.MonitorUserSecretName = dbcr.Name + "-proxysql-monitoruser"
	return nil
}

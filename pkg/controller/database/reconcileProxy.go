package database

import (
	"context"
	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	proxy "github.com/kloeckner-i/db-operator/pkg/utils/proxy"

	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileDatabase) createProxy(dbcr *kciv1alpha1.Database) error {
	if backend, _ := dbcr.GetBackendType(); backend != "google" {
		logrus.Infof("DB: namespace=%s, name=%s %s proxy creation is not yet implemented skipping...", dbcr.Namespace, dbcr.Name, backend)
		return nil
	}

	proxyInterface, err := determinProxyType(dbcr)
	if err != nil {
		return err
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
	logrus.Infof("DB: namespace=%s, name=%s proxy created", dbcr.Namespace, dbcr.Name)
	return nil
}

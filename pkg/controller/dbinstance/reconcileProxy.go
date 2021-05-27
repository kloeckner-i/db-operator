package dbinstance

import (
	"context"
	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	proxy "github.com/kloeckner-i/db-operator/pkg/utils/proxy"

	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileDbInstance) createProxy(dbin *kciv1alpha1.DbInstance) error {
	proxyInterface, err := determinProxyType(r.conf, dbin)
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
	err = r.client.Create(context.TODO(), deploy)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if resource already exists, update
			err = r.client.Update(context.TODO(), deploy)
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
	err = r.client.Create(context.TODO(), svc)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if resource already exists, update
			patch := client.MergeFrom(svc)
			err = r.client.Patch(context.TODO(), svc, patch)
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

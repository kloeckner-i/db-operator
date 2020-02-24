package database

import (
	"context"
	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	monitoring "github.com/kloeckner-i/db-operator/pkg/controller/database/monitoring"

	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileDatabase) createMonitoringExporter(dbcr *kciv1alpha1.Database) error {
	if !dbcr.Spec.Monitoring.Enable {
		// if not enabled, skip
		return nil
	}

	configmap, err := monitoring.ConfigMap(dbcr)
	if err != nil {
		return err
	}

	controllerutil.SetControllerReference(dbcr, configmap, r.scheme)
	err = r.client.Create(context.TODO(), configmap)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if resource already exists, update
			err = r.client.Update(context.TODO(), configmap)
			if err != nil {
				logrus.Errorf("DB: namespace=%s, name=%s failed updating monitoring exporter configmap", dbcr.Namespace, dbcr.Name)
				return err
			}
		} else {
			// failed to create
			logrus.Errorf("DB: namespace=%s, name=%s failed creating monitoring exporter configmap", dbcr.Namespace, dbcr.Name)
			return err
		}
	}

	deploy, err := monitoring.Deployment(dbcr)
	if err != nil {
		return err
	}

	controllerutil.SetControllerReference(dbcr, deploy, r.scheme)
	err = r.client.Create(context.TODO(), deploy)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if resource already exists, update
			err = r.client.Update(context.TODO(), deploy)
			if err != nil {
				logrus.Errorf("DB: namespace=%s, name=%s failed updating monitoring exporter deployment", dbcr.Namespace, dbcr.Name)
				return err
			}
		} else {
			// failed to create
			logrus.Errorf("DB: namespace=%s, name=%s failed creating monitoring exporter deployment", dbcr.Namespace, dbcr.Name)
			return err
		}
	}

	return nil
}

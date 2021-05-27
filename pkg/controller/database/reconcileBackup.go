package database

import (
	"context"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	backup "github.com/kloeckner-i/db-operator/pkg/controller/database/backup"

	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileDatabase) createBackupJob(dbcr *kciv1alpha1.Database) error {
	if !dbcr.Spec.Backup.Enable {
		// if not enabled, skip
		return nil
	}

	cronjob, err := backup.GCSBackupCron(r.conf, dbcr)
	if err != nil {
		return err
	}

	controllerutil.SetControllerReference(dbcr, cronjob, r.scheme)
	err = r.client.Create(context.TODO(), cronjob)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if resource already exists, update
			err = r.client.Update(context.TODO(), cronjob)
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

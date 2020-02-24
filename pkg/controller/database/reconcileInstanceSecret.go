package database

import (
	"context"
	"io/ioutil"
	"os"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"

	"github.com/kloeckner-i/db-operator/pkg/utils/kci"

	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

func (r *ReconcileDatabase) createInstanceAccessSecret(dbcr *kciv1alpha1.Database) error {
	if backend, _ := dbcr.GetBackendType(); backend != "google" {
		logrus.Debugf("DB: namespace=%s, name=%s %s doesn't need instance access secret skipping...", dbcr.Namespace, dbcr.Name, backend)
		return nil
	}

	data, err := ioutil.ReadFile(os.Getenv("GCSQL_CLIENT_CREDENTIALS"))
	if err != nil {
		return err
	}

	newName := GCSQLClientSecretName
	secretData := make(map[string][]byte)
	secretData["credentials.json"] = data
	newSecret := kci.SecretBuilder(newName, dbcr.GetNamespace(), secretData)

	err = r.client.Create(context.TODO(), newSecret)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if configmap resource already exists, update
			err = r.client.Update(context.TODO(), newSecret)
			if err != nil {
				logrus.Errorf("DB: namespace=%s, name=%s failed updating instance access secret", dbcr.Namespace, dbcr.Name)
				return err
			}
		} else {
			logrus.Errorf("DB: namespace=%s, name=%s failed creating instance access secret - %s", dbcr.Namespace, dbcr.Name, err)
			return err
		}
	}
	logrus.Infof("DB: namespace=%s, name=%s instance access secret created", dbcr.Namespace, dbcr.Name)
	return nil
}

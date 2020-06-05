package database

import (
	"context"
	"strconv"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *ReconcileDatabase) createInfoConfigMap(dbcr *kciv1alpha1.Database) error {
	instance, err := dbcr.GetInstanceRef()
	if err != nil {
		return err
	}

	info := instance.Status.DeepCopy().Info
	proxyStatus := dbcr.Status.ProxyStatus

	if proxyStatus.Status == true {
		info["DB_HOST"] = proxyStatus.ServiceName
		info["DB_PORT"] = strconv.FormatInt(int64(proxyStatus.SQLPort), 10)
	}

	databaseConfigResource := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dbcr.Namespace,
			Name:      dbcr.Spec.SecretName,
			Labels:    kci.BaseLabelBuilder(),
		},
		Data: info,
	}

	err = r.client.Create(context.TODO(), databaseConfigResource)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			// if configmap resource already exists, update
			err = r.client.Update(context.TODO(), databaseConfigResource)
			if err != nil {
				logrus.Errorf("DB: namespace=%s, name=%s failed updating database info configmap", dbcr.Namespace, dbcr.Name)
				return err
			}
		} else {
			logrus.Errorf("DB: namespace=%s, name=%s failed creating database info configmap", dbcr.Namespace, dbcr.Name)
			return err
		}
	}

	logrus.Infof("DB: namespace=%s, name=%s database info configmap created", dbcr.Namespace, dbcr.Name)
	return nil
}

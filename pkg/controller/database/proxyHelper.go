package database

import (
	"errors"
	"strconv"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	proxy "github.com/kloeckner-i/db-operator/pkg/utils/proxy"

	"github.com/sirupsen/logrus"
)

func determinProxyType(dbcr *kciv1alpha1.Database) (proxy.Proxy, error) {
	logrus.Debugf("DB: namespace=%s, name=%s - determinProxyType", dbcr.Namespace, dbcr.Name)
	backend, err := dbcr.GetBackendType()
	if err != nil {
		logrus.Errorf("could not get backend type %s - %s", dbcr.Name, err)
		return nil, err
	}

	instance, err := dbcr.GetInstanceRef()
	if err != nil {
		logrus.Errorf("can not create cloudsql proxy because can not get instanceRef - %s", err)
		return nil, err
	}

	engine, err := dbcr.GetEngineType()
	if err != nil {
		logrus.Errorf("can not create cloudsql proxy because can not get engineType - %s", err)
		return nil, err
	}

	portString := instance.Status.Info["DB_PORT"]
	port, err := strconv.Atoi(portString)
	if err != nil {
		logrus.Errorf("can not convert DB_PORT to int - %s", err)
		return nil, err
	}

	switch backend {
	case "google":
		labels := map[string]string{
			"app":     "cloudproxy",
			"db-name": dbcr.Name,
		}

		return &proxy.CloudProxy{
			NamePrefix:             "db-" + dbcr.Name,
			Namespace:              dbcr.Namespace,
			InstanceConnectionName: instance.Status.Info["DB_CONN"],
			AccessSecretName:       GCSQLClientSecretName,
			Engine:                 engine,
			Port:                   int32(port),
			Labels:                 kci.LabelBuilder(labels),
		}, nil
	case "percona":
		labels := map[string]string{
			"app":     "proxysql",
			"db-name": dbcr.Name,
		}

		return &proxy.ProxySQL{
			NamePrefix:            "db-" + dbcr.Name,
			Namespace:             dbcr.Namespace,
			Servers:               instance.Spec.Percona.ServerList,
			MaxConn:               instance.Spec.Percona.MaxConnection,
			UserSecretName:        dbcr.Spec.SecretName,
			MonitorUserSecretName: dbcr.Status.MonitorUserSecretName,
			Engine:                engine,
			Port:                  uint16(port),
			Labels:                kci.LabelBuilder(labels),
		}, nil
	default:
		err := errors.New("not supported backend type")
		return nil, err
	}
}

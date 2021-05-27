package database

import (
	"errors"
	"github.com/kloeckner-i/db-operator/pkg/config"
	"strconv"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	proxy "github.com/kloeckner-i/db-operator/pkg/utils/proxy"
	"github.com/kloeckner-i/db-operator/pkg/utils/proxy/proxysql"

	"github.com/sirupsen/logrus"
)

func determinProxyType(conf *config.Config, dbcr *kciv1alpha1.Database) (proxy.Proxy, error) {
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
			Conf:                   conf,
		}, nil
	case "percona":
		labels := map[string]string{
			"app":     "proxysql",
			"db-name": dbcr.Name,
		}

		var backends []proxysql.Backend
		for _, s := range instance.Spec.Percona.ServerList {
			backend := proxysql.Backend{
				Host:     s.Host,
				Port:     strconv.FormatInt(int64(s.Port), 10),
				MaxConn:  strconv.FormatInt(int64(s.MaxConnection), 10),
				ReadOnly: s.ReadOnly,
			}
			backends = append(backends, backend)
		}

		return &proxy.ProxySQL{
			NamePrefix:            "db-" + dbcr.Name,
			Namespace:             dbcr.Namespace,
			Servers:               backends,
			UserSecretName:        dbcr.Spec.SecretName,
			MonitorUserSecretName: dbcr.Status.MonitorUserSecretName,
			Engine:                engine,
			Labels:                kci.LabelBuilder(labels),
			Conf:                  conf,
		}, nil
	default:
		err := errors.New("not supported backend type")
		return nil, err
	}
}

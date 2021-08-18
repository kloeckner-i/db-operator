/*
 * Copyright 2021 kloeckner.i GmbH
 * Copyright 2018 The Operator-SDK Authors
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

package controllers

import (
	"errors"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/api/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/config"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	proxy "github.com/kloeckner-i/db-operator/pkg/utils/proxy"
	"github.com/kloeckner-i/db-operator/pkg/utils/proxy/proxysql"
	"github.com/sirupsen/logrus"
)

var (
	// ErrNoNamespace indicates that a namespace could not be found for the current
	// environment
	ErrNoNamespace = errors.New("namespace not found for current environment")
	// ErrNoProxySupport is thrown when proxy creation is not supported
	ErrNoProxySupport = errors.New("no proxy supported backend type")
)

func determineProxyTypeForDB(conf *config.Config, dbcr *kciv1alpha1.Database) (proxy.Proxy, error) {
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

func determineProxyTypeForInstance(conf *config.Config, dbin *kciv1alpha1.DbInstance) (proxy.Proxy, error) {
	logrus.Debugf("Instance: name=%s - determinProxyType", dbin.Name)
	operatorNamespace, err := getOperatorNamespace()
	if err != nil {
		// can not get operator namespace
		return nil, err
	}

	backend, err := dbin.GetBackendType()
	if err != nil {
		return nil, err
	}

	switch backend {
	case "google":
		portString := dbin.Status.Info["DB_PORT"]
		port, err := strconv.Atoi(portString)
		if err != nil {
			logrus.Errorf("can not convert DB_PORT to int - %s", err)
			return nil, err
		}

		labels := map[string]string{
			"app":           "cloudproxy",
			"instance-name": dbin.Name,
		}

		return &proxy.CloudProxy{
			NamePrefix:             "dbinstance-" + dbin.Name,
			Namespace:              operatorNamespace,
			InstanceConnectionName: dbin.Status.Info["DB_CONN"],
			AccessSecretName:       conf.Instances.Google.ClientSecretName,
			Engine:                 dbin.Spec.Engine,
			Port:                   int32(port),
			Labels:                 kci.LabelBuilder(labels),
			Conf:                   conf,
		}, nil
	default:
		return nil, ErrNoProxySupport
	}
}

// getOperatorNamespace returns the namespace the operator should be running in.
func getOperatorNamespace() (string, error) {
	nsBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNoNamespace
		}
		return "", err
	}
	return strings.TrimSpace(string(nsBytes)), nil
}

/*
 * Copyright 2021 kloeckner.i GmbH
 * Copyright 2018 The Operator-SDK Authors
 * Copyright 2023 The DB-Operator Authors
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

package proxy

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"

	kindav1beta1 "github.com/db-operator/db-operator/v2/api/v1beta1"
	"github.com/db-operator/db-operator/v2/pkg/config"
	"github.com/db-operator/db-operator/v2/pkg/utils/kci"
	proxy "github.com/db-operator/db-operator/v2/pkg/utils/proxy"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	// ErrNoNamespace indicates that a namespace could not be found for the current
	// environment
	ErrNoNamespace = errors.New("namespace not found for current environment")
	// ErrNoProxySupport is thrown when proxy creation is not supported
	ErrNoProxySupport = errors.New("no proxy supported backend type")
)

func DetermineProxyTypeForDB(ctx context.Context, conf *config.Config, dbcr *kindav1beta1.Database, instance *kindav1beta1.DbInstance) (proxy.Proxy, error) {
	log := log.FromContext(ctx)
	log.V(2).Info("Database determineProxyType", "namespace", dbcr.Namespace, "name", dbcr.Name)
	backend, err := instance.GetBackendType()
	if err != nil {
		log.Error(err, "could not get backend type", dbcr.Name)
		return nil, err
	}

	portString := instance.Status.Info["DB_PORT"]
	port, err := strconv.ParseUint(portString, 10, 16)
	if err != nil {
		log.Error(err, "can not convert DB_PORT to int")
		return nil, err
	}

	switch backend {
	case "google":
		labels := map[string]string{
			"app":     "cloudproxy",
			"db-name": dbcr.Name,
		}

		monitoringEnabled := instance.IsMonitoringEnabled()

		proxy := &proxy.CloudProxy{
			NamePrefix:             "db-" + dbcr.Name,
			Namespace:              dbcr.Namespace,
			InstanceConnectionName: instance.Status.Info["DB_CONN"],
			AccessSecretName:       dbcr.InstanceAccessSecretName(),
			Engine:                 dbcr.Status.Engine,
			Port:                   int32(port),
			Labels:                 kci.LabelBuilder(labels),
			Conf:                   conf,
			MonitoringEnabled:      monitoringEnabled,
		}
		return proxy, nil

	default:
		err := errors.New("not supported backend type")
		return nil, err
	}
}

func DetermineProxyTypeForInstance(ctx context.Context, conf *config.Config, dbin *kindav1beta1.DbInstance) (proxy.Proxy, error) {
	log := log.FromContext(ctx)
	log.V(2).Info("Instance determineProxyType", "name", dbin.Name)
	operatorNamespace, err := GetOperatorNamespace()
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
		port, err := strconv.ParseInt(portString, 10, 32)
		if err != nil {
			log.Error(err, "can not convert DB_PORT to int")
			return nil, err
		}

		labels := map[string]string{
			"app":           "cloudproxy",
			"instance-name": dbin.Name,
		}

		monitoringEnabled := dbin.IsMonitoringEnabled()

		var accessSecretName string
		if dbin.Spec.Google.ClientSecret.Name != "" {
			accessSecretName = dbin.Spec.Google.ClientSecret.Name
		} else {
			accessSecretName = conf.Instances.Google.ClientSecretName
		}

		return &proxy.CloudProxy{
			NamePrefix:             "dbinstance-" + dbin.Name,
			Namespace:              operatorNamespace,
			InstanceConnectionName: dbin.Status.Info["DB_CONN"],
			AccessSecretName:       accessSecretName,
			Engine:                 dbin.Spec.Engine,
			Port:                   int32(port),
			Labels:                 kci.LabelBuilder(labels),
			Conf:                   conf,
			MonitoringEnabled:      monitoringEnabled,
		}, nil
	default:
		return nil, ErrNoProxySupport
	}
}

// getOperatorNamespace returns the namespace the operator should be running in.
func GetOperatorNamespace() (string, error) {
	nsBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNoNamespace
		}
		return "", err
	}
	return strings.TrimSpace(string(nsBytes)), nil
}

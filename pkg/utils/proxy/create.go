/*
 * Copyright 2021 kloeckner.i GmbH
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

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// BuildDeployment builds kubernetes deployment object to create proxy container of the database
func BuildDeployment(ctx context.Context, proxy Proxy) (*v1apps.Deployment, error) {
	log := log.FromContext(ctx)
	deploy, err := proxy.buildDeployment()
	if err != nil {
		log.Error(err, "failed building proxy deployment")
		return nil, err
	}

	return deploy, nil
}

// BuildService builds kubernetes service object for proxy service of the database
func BuildService(ctx context.Context, proxy Proxy) (*v1.Service, error) {
	log := log.FromContext(ctx)
	svc, err := proxy.buildService()
	if err != nil {
		log.Error(err, "failed building proxy service")
		return nil, err
	}

	return svc, nil
}

// BuildConfigmap builds kubernetes configmap object used by proxy container of the database
func BuildConfigmap(ctx context.Context, proxy Proxy) (*v1.ConfigMap, error) {
	log := log.FromContext(ctx)
	cm, err := proxy.buildConfigMap()
	if err != nil {
		log.Error(err, "failed building proxy configmap")
		return nil, err
	}

	return cm, nil
}

// BuildServiceMonitor builds kubernetes prometheus ServiceMonitor CR object used for monitoring
func BuildServiceMonitor(ctx context.Context, proxy Proxy) (*promv1.ServiceMonitor, error) {
	log := log.FromContext(ctx)
	promSerMon, err := proxy.buildServiceMonitor()
	if err != nil {
		log.Error(err, "failed building promServiceMonitor configmap")
		return nil, err
	}

	return promSerMon, nil
}

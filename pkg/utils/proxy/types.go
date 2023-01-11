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
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Proxy for database
type Proxy interface {
	buildDeployment(ownership []metav1.OwnerReference) (*v1apps.Deployment, error)
	buildService(ownership []metav1.OwnerReference) (*v1.Service, error)
	buildServiceMonitor(ownership []metav1.OwnerReference) (*promv1.ServiceMonitor, error)
	buildConfigMap(ownership []metav1.OwnerReference) (*v1.ConfigMap, error)
}

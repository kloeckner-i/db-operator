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

package monitoring

import (
	"fmt"
	"github.com/kloeckner-i/db-operator/pkg/config"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/api/v1alpha1"

	v1apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

// Deployment builds kubernetes deployment object
// to run prometheus exporter to expose dbcr metrics
func Deployment(conf *config.Config, dbcr *kciv1alpha1.Database) (*v1apps.Deployment, error) {
	engine, err := dbcr.GetEngineType()
	if err != nil {
		return nil, err
	}

	switch engine {
	case "postgres":
		return pgDeployment(conf, dbcr)
	default:
		return nil, fmt.Errorf("Monitoring: db engine monitoring for %s not implemented", engine)
	}
}

// ConfigMap builds kubernetes configmap object
// which contains query to execute by prometheus exporter
func ConfigMap(conf *config.Config, dbcr *kciv1alpha1.Database) (*v1.ConfigMap, error) {
	engine, err := dbcr.GetEngineType()
	if err != nil {
		return nil, err
	}

	switch engine {
	case "postgres":
		return pgExporterQueryCM(conf, dbcr)
	default:
		return nil, fmt.Errorf("monitoring: exporter query creation for db engine %s not implemented", engine)
	}

}

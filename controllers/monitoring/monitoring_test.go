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
	"github.com/kloeckner-i/db-operator/pkg/config"
	"testing"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/api/v1alpha1"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
	v1apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

func TestPGDeployment(t *testing.T) {
	dbcr := &kciv1alpha1.Database{}

	// mock PGDeployment
	pgDeploymentMock := func(*config.Config, *kciv1alpha1.Database) (*v1apps.Deployment, error) {
		return &v1apps.Deployment{}, nil
	}
	patch := monkey.Patch(pgDeployment, pgDeploymentMock)
	defer patch.Unpatch()

	res, err := pgDeployment(&config.Config{}, dbcr)
	assert.Equal(t, &v1apps.Deployment{}, res)
	assert.Equal(t, nil, err)
}

func TestPGExporterQueryCM(t *testing.T) {
	dbcr := &kciv1alpha1.Database{}

	// mock PGDeployment
	pgExporterQueryCMMock := func(*config.Config, *kciv1alpha1.Database) (*v1.ConfigMap, error) {
		return &v1.ConfigMap{}, nil
	}
	patch := monkey.Patch(pgExporterQueryCM, pgExporterQueryCMMock)
	defer patch.Unpatch()

	res, err := pgExporterQueryCM(&config.Config{}, dbcr)
	assert.Equal(t, &v1.ConfigMap{}, res)
	assert.Equal(t, nil, err)
}

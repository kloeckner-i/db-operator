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

package proxy_test

import (
	"testing"

	kindav1beta1 "github.com/db-operator/db-operator/v2/api/v1beta1"
	proxyhelper "github.com/db-operator/db-operator/v2/internal/helpers/proxy"
	"github.com/db-operator/db-operator/v2/internal/utils/testutils"
	"github.com/db-operator/db-operator/v2/pkg/config"
	"github.com/db-operator/db-operator/v2/pkg/utils/proxy"
	"github.com/stretchr/testify/assert"
)

func makeGsqlInstance() kindav1beta1.DbInstance {
	info := make(map[string]string)
	info["DB_CONN"] = "test-conn"
	info["DB_PORT"] = "1234"
	dbInstance := kindav1beta1.DbInstance{
		Spec: kindav1beta1.DbInstanceSpec{
			DbInstanceSource: kindav1beta1.DbInstanceSource{
				Google: &kindav1beta1.GoogleInstance{},
			},
		},
		Status: kindav1beta1.DbInstanceStatus{
			Info: info,
		},
	}
	return dbInstance
}

func makeGenericInstance() kindav1beta1.DbInstance {
	info := make(map[string]string)
	info["DB_CONN"] = "test-conn"
	info["DB_PORT"] = "1234"
	dbInstance := kindav1beta1.DbInstance{
		Spec: kindav1beta1.DbInstanceSpec{
			DbInstanceSource: kindav1beta1.DbInstanceSource{
				Generic: &kindav1beta1.GenericInstance{},
			},
		},
		Status: kindav1beta1.DbInstanceStatus{
			Info: info,
		},
	}
	return dbInstance
}

func mockOperatorNamespace() (string, error) {
	return "operator", nil
}

func TestUnitDetermineProxyTypeForDBGoogleBackend(t *testing.T) {
	config := &config.Config{}
	dbin := makeGsqlInstance()
	db := testutils.NewPostgresTestDbCr(dbin)
	dbProxy, err := proxyhelper.DetermineProxyTypeForDB(t.Context(), config, db, &dbin)
	assert.NoError(t, err)
	cloudProxy, ok := dbProxy.(*proxy.CloudProxy)
	assert.Equal(t, ok, true, "expected true")
	assert.Equal(t, cloudProxy.AccessSecretName, db.InstanceAccessSecretName())
}

func TestUnitDetermineProxyTypeForDBGenericBackend(t *testing.T) {
	config := &config.Config{}
	dbin := makeGenericInstance()
	db := testutils.NewPostgresTestDbCr(dbin)
	_, err := proxyhelper.DetermineProxyTypeForDB(t.Context(), config, db, &dbin)
	assert.Error(t, err)
}

func TestUnitDetermineProxyTypeForGenericInstance(t *testing.T) {
	config := &config.Config{}
	dbin := makeGenericInstance()
	_, err := proxyhelper.DetermineProxyTypeForInstance(t.Context(), config, &dbin)
	assert.Error(t, err)
}

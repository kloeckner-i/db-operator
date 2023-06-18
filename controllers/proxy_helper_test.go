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

package controllers

import (
	"os"
	"testing"

	"bou.ke/monkey"
	kindav1beta1 "github.com/db-operator/db-operator/api/v1beta1"
	"github.com/db-operator/db-operator/pkg/config"
	"github.com/db-operator/db-operator/pkg/utils/proxy"
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

func TestDetermineProxyTypeForDBGoogleBackend(t *testing.T) {
	config := &config.Config{}
	dbin := makeGsqlInstance()
	db := newPostgresTestDbCr(dbin)
	dbProxy, err := determineProxyTypeForDB(config, db)
	assert.NoError(t, err)
	cloudProxy, ok := dbProxy.(*proxy.CloudProxy)
	assert.Equal(t, ok, true, "expected true")
	assert.Equal(t, cloudProxy.AccessSecretName, db.InstanceAccessSecretName())
}

func TestDetermineProxyTypeForDBGenericBackend(t *testing.T) {
	config := &config.Config{}
	dbin := makeGenericInstance()
	db := newPostgresTestDbCr(dbin)
	_, err := determineProxyTypeForDB(config, db)
	assert.Error(t, err)
}

func TestDetermineProxyTypeForGoogleInstance(t *testing.T) {
	os.Setenv("CONFIG_PATH", "../pkg/config/test/config_ok.yaml")
	config := config.LoadConfig()
	dbin := makeGsqlInstance()
	patchGetOperatorNamespace := monkey.Patch(getOperatorNamespace, mockOperatorNamespace)
	defer patchGetOperatorNamespace.Unpatch()
	dbProxy, err := determineProxyTypeForInstance(&config, &dbin)
	assert.NoError(t, err)
	cloudProxy, ok := dbProxy.(*proxy.CloudProxy)
	assert.Equal(t, ok, true, "expected true")
	assert.Equal(t, cloudProxy.AccessSecretName, "cloudsql-readonly-serviceaccount")

	dbin.Spec.Google.ClientSecret.Name = "test-client-secret"
	dbProxy, err = determineProxyTypeForInstance(&config, &dbin)
	assert.NoError(t, err)
	cloudProxy, ok = dbProxy.(*proxy.CloudProxy)
	assert.Equal(t, ok, true, "expected true")
	assert.Equal(t, cloudProxy.AccessSecretName, "test-client-secret")
}

func TestDetermineProxyTypeForGenericInstance(t *testing.T) {
	config := &config.Config{}
	dbin := makeGenericInstance()
	_, err := determineProxyTypeForInstance(config, &dbin)
	assert.Error(t, err)
}

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

package kci

import (
	"github.com/kloeckner-i/db-operator/api/v1alpha1"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigMapBuilder(t *testing.T) {
	name := "test-configmap"
	om := metav1.ObjectMeta{Namespace: "TestNS"}
	s := v1alpha1.DatabaseSpec{SecretName: "TestSec"}
	owner := v1alpha1.Database{ObjectMeta: om, Spec: s}
	data := map[string]string{
		"key": "value",
	}

	configmap := ConfigMapBuilder(name, owner.Namespace, data)

	assert.Equal(t, owner.Namespace, configmap.GetNamespace(), "Namespace has not expected Value")
	assert.Equal(t, data, configmap.Data, "Config Name not match expected Value")
}

func TestSecretBuilder(t *testing.T) {
	name := "test-secret"
	o := metav1.ObjectMeta{Namespace: "TestNS"}
	owner := v1alpha1.Database{ObjectMeta: o}
	data := map[string][]byte{
		"key": []byte("secret"),
	}

	secret := SecretBuilder(name, owner.Namespace, data)

	assert.Equal(t, owner.Namespace, secret.GetNamespace(), "Namespace has not expected Value")
	assert.Equal(t, data, secret.Data, "Secret Data not match expected Value")
}

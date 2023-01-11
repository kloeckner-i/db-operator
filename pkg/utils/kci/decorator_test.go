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
	"testing"

	"github.com/kloeckner-i/db-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigMapBuilder(t *testing.T) {
	name := "test-configmap"
	ownership := []metav1.OwnerReference{}
	om := metav1.ObjectMeta{Namespace: "TestNS"}
	s := v1alpha1.DatabaseSpec{SecretName: "TestSec"}
	owner := v1alpha1.Database{ObjectMeta: om, Spec: s}
	data := map[string]string{
		"key": "value",
	}

	configmap := ConfigMapBuilder(name, owner.Namespace, data, ownership)

	assert.Equal(t, owner.Namespace, configmap.GetNamespace(), "Namespace has not expected Value")
	assert.Equal(t, data, configmap.Data, "Config Name not match expected Value")
	assert.Equal(t, len(configmap.OwnerReferences), 0, "Unexpected size of an OwnerReference")
}

func TestConfigMapBuilderWithOwnerReference(t *testing.T) {
	name := "test-configmap"
	ownership := []metav1.OwnerReference{}
	ownership = append(ownership, metav1.OwnerReference{
		APIVersion: "api-version",
		Kind: "kind",
		Name: "name",
		UID: "uid",
	})
	om := metav1.ObjectMeta{Namespace: "TestNS"}
	s := v1alpha1.DatabaseSpec{SecretName: "TestSec"}
	owner := v1alpha1.Database{ObjectMeta: om, Spec: s}
	data := map[string]string{
		"key": "value",
	}

	configmap := ConfigMapBuilder(name, owner.Namespace, data, ownership)

	assert.Equal(t, owner.Namespace, configmap.GetNamespace(), "Namespace has not expected Value")
	assert.Equal(t, data, configmap.Data, "Config Name not match expected Value")
	assert.Equal(t, len(configmap.OwnerReferences), 1, "Unexpected size of an OwnerReference")
	assert.Equal(t, configmap.OwnerReferences[0].APIVersion, ownership[0].APIVersion, "API Version in the OwnerReference is wrong")
	assert.Equal(t, configmap.OwnerReferences[0].Kind, ownership[0].Kind, "Kind in the OwnerReference is wrong")
	assert.Equal(t, configmap.OwnerReferences[0].Name, ownership[0].Name, "Name in the OwnerReference is wrong")
	assert.Equal(t, configmap.OwnerReferences[0].UID, ownership[0].UID, "UID in the OwnerReference is wrong")
}

func TestSecretBuilder(t *testing.T) {
	name := "test-secret"
	o := metav1.ObjectMeta{Namespace: "TestNS"}
	ownership := []metav1.OwnerReference{}

	owner := v1alpha1.Database{ObjectMeta: o}
	data := map[string][]byte{
		"key": []byte("secret"),
	}

	secret := SecretBuilder(name, owner.Namespace, data, ownership)

	assert.Equal(t, owner.Namespace, secret.GetNamespace(), "Namespace has not expected Value")
	assert.Equal(t, data, secret.Data, "Secret Data not match expected Value")
	assert.Equal(t, len(secret.OwnerReferences), 0, "Unexpected size of an OwnerReference")
}

func TestSecretBuilderWithOwnerReference(t *testing.T) {
	name := "test-secret"
	o := metav1.ObjectMeta{Namespace: "TestNS"}
	ownership := []metav1.OwnerReference{}

	ownership = append(ownership, metav1.OwnerReference{
		APIVersion: "api-version",
		Kind: "kind",
		Name: "name",
		UID: "uid",
	})

	owner := v1alpha1.Database{ObjectMeta: o}
	data := map[string][]byte{
		"key": []byte("secret"),
	}

	secret := SecretBuilder(name, owner.Namespace, data, ownership)

	assert.Equal(t, owner.Namespace, secret.GetNamespace(), "Namespace has not expected Value")
	assert.Equal(t, data, secret.Data, "Secret Data not match expected Value")
	assert.Equal(t, len(secret.OwnerReferences), 1, "Unexpected size of an OwnerReference")
	assert.Equal(t, secret.OwnerReferences[0].APIVersion, ownership[0].APIVersion, "API Version in the OwnerReference is wrong")
	assert.Equal(t, secret.OwnerReferences[0].Kind, ownership[0].Kind, "Kind in the OwnerReference is wrong")
	assert.Equal(t, secret.OwnerReferences[0].Name, ownership[0].Name, "Name in the OwnerReference is wrong")
	assert.Equal(t, secret.OwnerReferences[0].UID, ownership[0].UID, "UID in the OwnerReference is wrong")
}

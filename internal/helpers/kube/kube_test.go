/*
 * Copyright 2023 DB-Operator Authors
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

package kube_test

import (
	"testing"

	kindav1beta1 "github.com/db-operator/db-operator/v2/api/v1beta1"
	"github.com/db-operator/db-operator/v2/internal/helpers/kube"
	"github.com/db-operator/db-operator/v2/pkg/consts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "default",
		},
	}

	configmap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap",
			Namespace: "default",
		},
	}

	database = &kindav1beta1.Database{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Database",
			APIVersion: "kinda.rocks/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "database",
			Namespace: "default",
			UID:       types.UID("database"),
		},
	}
)

func TestDeleteUsedByLabelsSet(t *testing.T) {
	secretCopy := secret.DeepCopy()
	kh := &kube.KubeHelper{Caller: database.DeepCopy()}

	usedByLabels := map[string]string{
		consts.USED_BY_KIND_LABEL_KEY: database.Kind,
		consts.USED_BY_NAME_LABEL_KEY: database.Name,
	}
	secretCopy.SetLabels(usedByLabels)

	actual := kh.DeleteUsedByLabels(secretCopy)
	assert.NotContains(t, actual.GetLabels(), consts.USED_BY_KIND_LABEL_KEY)
	assert.NotContains(t, actual.GetLabels(), consts.USED_BY_NAME_LABEL_KEY)
}

func TestDeleteUsedByLabelsUnset(t *testing.T) {
	secretCopy := secret.DeepCopy()
	kh := &kube.KubeHelper{Caller: database.DeepCopy()}

	actual := kh.DeleteUsedByLabels(secretCopy)
	assert.NotContains(t, actual.GetLabels(), consts.USED_BY_KIND_LABEL_KEY)
	assert.NotContains(t, actual.GetLabels(), consts.USED_BY_NAME_LABEL_KEY)
}

func TestDeleteUsedByLabelsAdditional(t *testing.T) {
	secretCopy := secret.DeepCopy()
	kh := &kube.KubeHelper{Caller: database.DeepCopy()}
	usedByLabels := map[string]string{
		consts.USED_BY_KIND_LABEL_KEY: database.Kind,
		consts.USED_BY_NAME_LABEL_KEY: database.Name,
		"test":                        "test",
	}
	secretCopy.SetLabels(usedByLabels)

	actual := kh.DeleteUsedByLabels(secretCopy)
	assert.NotContains(t, actual.GetLabels(), consts.USED_BY_KIND_LABEL_KEY)
	assert.NotContains(t, actual.GetLabels(), consts.USED_BY_NAME_LABEL_KEY)
	assert.Contains(t, actual.GetLabels(), "test")
	assert.Equal(t, actual.GetLabels()["test"], "test")
}

func TestDeleteUsedByLabelsExternal(t *testing.T) {
	kh := &kube.KubeHelper{Caller: database.DeepCopy()}
	secretCopy := secret.DeepCopy()
	usedByLabels := map[string]string{
		"test": "test",
	}
	secretCopy.SetLabels(usedByLabels)

	actual := kh.DeleteUsedByLabels(secretCopy)
	assert.NotContains(t, actual.GetLabels(), consts.USED_BY_KIND_LABEL_KEY)
	assert.NotContains(t, actual.GetLabels(), consts.USED_BY_NAME_LABEL_KEY)
	assert.Contains(t, actual.GetLabels(), "test")
	assert.Equal(t, actual.GetLabels()["test"], "test")
}

func TestBuildOwnerReferenceCleanup(t *testing.T) {
	databaseCopy := database.DeepCopy()
	databaseCopy.Spec.Cleanup = true
	kh := &kube.KubeHelper{Caller: databaseCopy}
	actual := kh.BuildOwnerReference()
	expected := metav1.OwnerReference{
		APIVersion: database.APIVersion,
		Kind:       database.Kind,
		Name:       database.Name,
		UID:        database.UID,
	}
	assert.Equal(t, expected, actual)
}

func TestBuildOwnerReferenceNotCleanup(t *testing.T) {
	databaseCopy := database.DeepCopy()
	databaseCopy.Spec.Cleanup = false
	kh := &kube.KubeHelper{Caller: databaseCopy}
	actual := kh.BuildOwnerReference()
	expected := metav1.OwnerReference{}
	assert.Equal(t, expected, actual)
}

func TestSetOwnerReferenceCleanupOnly(t *testing.T) {
	databaseCopy := database.DeepCopy()
	databaseCopy.Spec.Cleanup = true
	kh := &kube.KubeHelper{Caller: databaseCopy}
	secretCopy := secret.DeepCopy()
	ownerRef := kh.BuildOwnerReference()
	actual := kh.SetOwnerReference(secretCopy, ownerRef)

	expected := []metav1.OwnerReference{
		{
			APIVersion: database.APIVersion,
			Kind:       database.Kind,
			Name:       database.Name,
			UID:        database.UID,
		},
	}

	assert.Equal(t, expected, actual.GetOwnerReferences())
}

func TestSetOwnerReferenceCleanupExtra(t *testing.T) {
	databaseCopy := database.DeepCopy()
	databaseCopy.Spec.Cleanup = true
	kh := &kube.KubeHelper{Caller: databaseCopy}
	secretCopy := secret.DeepCopy()

	existingOwnerRef := []metav1.OwnerReference{
		{
			APIVersion: "test",
			Kind:       "test",
			Name:       "test",
			UID:        types.UID("test"),
		},
	}

	secretCopy.SetOwnerReferences(existingOwnerRef)

	ownerRef := kh.BuildOwnerReference()
	actual := kh.SetOwnerReference(secretCopy, ownerRef)

	expected := append(existingOwnerRef, metav1.OwnerReference{
		APIVersion: database.APIVersion,
		Kind:       database.Kind,
		Name:       database.Name,
		UID:        database.UID,
	})

	assert.Equal(t, expected, actual.GetOwnerReferences())
}

func TestSetOwnerReferenceNotCleanupOnly(t *testing.T) {
	databaseCopy := database.DeepCopy()
	databaseCopy.Spec.Cleanup = false
	kh := &kube.KubeHelper{Caller: databaseCopy}
	secretCopy := secret.DeepCopy()

	ownerRef := kh.BuildOwnerReference()
	actual := kh.SetOwnerReference(secretCopy, ownerRef)

	assert.Equal(t, []metav1.OwnerReference{}, actual.GetOwnerReferences())
}

func TestSetOwnerReferenceNotCleanupExtra(t *testing.T) {
	databaseCopy := database.DeepCopy()
	databaseCopy.Spec.Cleanup = false
	kh := &kube.KubeHelper{Caller: databaseCopy}
	secretCopy := secret.DeepCopy()

	existingOwnerRef := []metav1.OwnerReference{
		{
			APIVersion: "test",
			Kind:       "test",
			Name:       "test",
			UID:        types.UID("test"),
		},
	}

	secretCopy.SetOwnerReferences(existingOwnerRef)

	ownerRef := kh.BuildOwnerReference()
	actual := kh.SetOwnerReference(secretCopy, ownerRef)

	expected := existingOwnerRef

	assert.Equal(t, expected, actual.GetOwnerReferences())
}

func TestBuildUsedByLabels(t *testing.T) {
	kh := &kube.KubeHelper{Caller: database}
	actual := kh.BuildUsedByLabels()
	expected := map[string]string{
		consts.USED_BY_KIND_LABEL_KEY: database.GetObjectKind().GroupVersionKind().Kind,
		consts.USED_BY_NAME_LABEL_KEY: database.GetName(),
	}
	assert.Equal(t, expected, actual)
}

func TestSetUsedByLabelsOnly(t *testing.T) {
	secretCopy := secret.DeepCopy()
	kh := &kube.KubeHelper{Caller: database}
	actual := kh.SetUsedByLabels(secretCopy)
	expected := map[string]string{
		consts.USED_BY_KIND_LABEL_KEY: database.GetObjectKind().GroupVersionKind().Kind,
		consts.USED_BY_NAME_LABEL_KEY: database.GetName(),
	}
	assert.Equal(t, expected, actual.GetLabels())
}

func TestSetUsedByLabelsExtre(t *testing.T) {
	secretCopy := secret.DeepCopy()
	secretCopy.Labels = map[string]string{
		"test": "test",
	}
	kh := &kube.KubeHelper{Caller: database}
	actual := kh.SetUsedByLabels(secretCopy)
	expected := map[string]string{
		consts.USED_BY_KIND_LABEL_KEY: database.GetObjectKind().GroupVersionKind().Kind,
		consts.USED_BY_NAME_LABEL_KEY: database.GetName(),
		"test":                        "test",
	}
	assert.Equal(t, expected, actual.GetLabels())
}

func TestIsUsedByCallerYes(t *testing.T) {
	kh := &kube.KubeHelper{Caller: database}
	secretCopy := secret.DeepCopy()
	usedByLabels := map[string]string{
		consts.USED_BY_KIND_LABEL_KEY: database.Kind,
		consts.USED_BY_NAME_LABEL_KEY: database.Name,
	}
	secretCopy.SetLabels(usedByLabels)
	assert.True(t, kh.IsUsedByCaller(secretCopy))
}

func TestIsUsedByCallerNoKind(t *testing.T) {
	kh := &kube.KubeHelper{Caller: database}
	secretCopy := secret.DeepCopy()
	usedByLabels := map[string]string{
		consts.USED_BY_KIND_LABEL_KEY: "test",
		consts.USED_BY_NAME_LABEL_KEY: database.Name,
	}
	secretCopy.SetLabels(usedByLabels)
	assert.False(t, kh.IsUsedByCaller(secretCopy))
}

func TestIsUsedByCallerNoName(t *testing.T) {
	kh := &kube.KubeHelper{Caller: database}
	secretCopy := secret.DeepCopy()
	usedByLabels := map[string]string{
		consts.USED_BY_KIND_LABEL_KEY: database.Kind,
		consts.USED_BY_NAME_LABEL_KEY: "test",
	}
	secretCopy.SetLabels(usedByLabels)
	assert.False(t, kh.IsUsedByCaller(secretCopy))
}

func TestIsUsedByCallerNoBoth(t *testing.T) {
	kh := &kube.KubeHelper{Caller: database}
	secretCopy := secret.DeepCopy()
	usedByLabels := map[string]string{
		consts.USED_BY_KIND_LABEL_KEY: "test",
		consts.USED_BY_NAME_LABEL_KEY: "test",
	}
	secretCopy.SetLabels(usedByLabels)
	assert.False(t, kh.IsUsedByCaller(secretCopy))
}

func TestIsUsedByAnyYes(t *testing.T) {
	kh := &kube.KubeHelper{}
	secretCopy := secret.DeepCopy()
	usedByLabels := map[string]string{
		consts.USED_BY_KIND_LABEL_KEY: database.Kind,
	}
	secretCopy.SetLabels(usedByLabels)
	assert.True(t, kh.IsUsedByAny(secretCopy))
}

func TestIsUsedByAnyNo(t *testing.T) {
	kh := &kube.KubeHelper{}
	secretCopy := secret.DeepCopy()
	assert.False(t, kh.IsUsedByAny(secretCopy))
}

func TestGetValueByKeyString(t *testing.T) {
	data := map[string]string{
		"test": "test",
	}
	val, ok := kube.GetValueByKey(data, "test")
	assert.True(t, ok)
	assert.Equal(t, "test", val)
}

func TestGetValueByKeyStringNotFound(t *testing.T) {
	data := map[string]string{
		"test": "test",
	}
	val, ok := kube.GetValueByKey(data, "dummy")
	assert.False(t, ok)
	assert.Equal(t, "", val)
}

func TestGetValueByKeyBytes(t *testing.T) {
	data := map[string][]byte{
		"test": []byte("test"),
	}
	val, ok := kube.GetValueByKey(data, "test")
	assert.True(t, ok)
	assert.Equal(t, []byte("test"), val)
}

func TestGetValueByKeybyteNotFound(t *testing.T) {
	data := map[string][]byte{
		"test": []byte("test"),
	}
	val, ok := kube.GetValueByKey(data, "dummy")
	assert.False(t, ok)
	assert.Equal(t, []byte(nil), val)
}

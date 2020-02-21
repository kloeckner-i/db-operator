package kci

import (
	"testing"

	"github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"

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

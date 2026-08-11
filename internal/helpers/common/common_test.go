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

package common_test

import (
	"context"
	"testing"

	kindav1beta1 "github.com/db-operator/db-operator/v2/api/v1beta1"
	"github.com/db-operator/db-operator/v2/internal/helpers/common"
	"github.com/db-operator/db-operator/v2/internal/utils/testutils"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUnitIsDBChangedPostgres(t *testing.T) {
	db := testutils.NewPostgresTestDbCr(testutils.NewPostgresTestDbInstanceCr())

	testDbSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: testutils.TestNamespace, Name: testutils.TestSecretName},
		Data: map[string][]byte{
			"POSTGRES_DB":       []byte("testdb"),
			"POSTGRES_USER":     []byte("testuser"),
			"POSTGRES_PASSWORD": []byte("testpassword"),
		},
	}

	common.AddDBChecksum(db, testDbSecret)
	assert.Equal(t, "11535687189846980805", db.Annotations["checksum/secret"])
	nochange := common.IsDBChanged(db, testDbSecret)
	assert.Equal(t, nochange, false, "expected false")

	testDbSecret.Data["POSTGRES_PASSWORD"] = []byte("testpasswordNEW")

	change := common.IsDBChanged(db, testDbSecret)
	assert.Equal(t, change, true, "expected true")
}

func TestUnitIsDBChangedMysql(t *testing.T) {
	db := testutils.NewPostgresTestDbCr(testutils.NewPostgresTestDbInstanceCr())

	testDbSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: testutils.TestNamespace, Name: testutils.TestSecretName},
		Data: map[string][]byte{
			"DB":       []byte("testdb"),
			"USER":     []byte("testuser"),
			"PASSWORD": []byte("testpassword"),
		},
	}

	common.AddDBChecksum(db, testDbSecret)
	assert.Equal(t, "5437193145901086135", db.Annotations["checksum/secret"])
	nochange := common.IsDBChanged(db, testDbSecret)
	assert.Equal(t, nochange, false, "expected false")

	testDbSecret.Data["POSTGRES_PASSWORD"] = []byte("testpasswordNEW")

	change := common.IsDBChanged(db, testDbSecret)
	assert.Equal(t, change, true, "expected true")
}

func TestUnitIsDBChangedExtraKeys(t *testing.T) {
	db := testutils.NewPostgresTestDbCr(testutils.NewPostgresTestDbInstanceCr())

	testDbSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: testutils.TestNamespace, Name: testutils.TestSecretName},
		Data: map[string][]byte{
			"POSTGRES_DB":       []byte("testdb"),
			"POSTGRES_USER":     []byte("testuser"),
			"POSTGRES_PASSWORD": []byte("testpassword"),
			"DB":                []byte("testdb"),
			"USER":              []byte("testuser"),
			"PASSWORD":          []byte("testpassword"),
			"TEST_1":            []byte("Iwillchange"),
		},
	}

	common.AddDBChecksum(db, testDbSecret)
	testDbSecret.Data["TEST_1"] = []byte("Ihavechanged")

	nochange := common.IsDBChanged(db, testDbSecret)
	assert.Equal(t, nochange, false, "expected false")
}

func TestUnitGenerateDBInstanceChecksums(t *testing.T) {
	ctx := context.Background()

	t.Run("Basic Spec and AdminSecret", func(t *testing.T) {
		// Test case: Basic instance with only AdminSecret referenced.
		// Verifies that 'spec' and 'adminSecret' keys are populated.
		dbin := &kindav1beta1.DbInstance{
			Spec: kindav1beta1.DbInstanceSpec{
				Engine: "postgres",
			},
		}
		data := common.DbInstanceData{
			AdminSecret: &corev1.Secret{
				Data: map[string][]byte{"password": []byte("pass")},
			},
		}

		checksums := common.GenerateDBInstanceChecksums(dbin, data)
		assert.Contains(t, checksums, "spec")
		assert.Contains(t, checksums, "adminSecret")
		assert.NotContains(t, checksums, "config")
	})

	t.Run("Google Backend with ConfigMap and ClientSecret", func(t *testing.T) {
		// Test case: Google backend with both ConfigMap and ClientSecret.
		// Verifies that backend-specific keys are correctly included.
		dbin := &kindav1beta1.DbInstance{
			Spec: kindav1beta1.DbInstanceSpec{
				Engine: "postgres",
				DbInstanceSource: kindav1beta1.DbInstanceSource{
					Google: &kindav1beta1.GoogleInstance{
						ClientSecret: kindav1beta1.NamespacedName{Name: "client-sec"},
					},
				},
			},
		}
		data := common.DbInstanceData{
			ConfigMap: &corev1.ConfigMap{
				Data: map[string]string{"config": "val"},
			},
			ClientSecret: &corev1.Secret{
				Data: map[string][]byte{"key": []byte("val")},
			},
		}

		checksums := common.GenerateDBInstanceChecksums(dbin, data)
		assert.Contains(t, checksums, "config")
		assert.Contains(t, checksums, "clientSecret")
	})

	t.Run("Generic Backend with Host, Port and PublicIP FromRef", func(t *testing.T) {
		// Test case: Generic backend with all fields fetched from external references.
		// Verifies that generic-specific 'From' keys are included.
		dbin := &kindav1beta1.DbInstance{
			Spec: kindav1beta1.DbInstanceSpec{
				Engine: "postgres",
				DbInstanceSource: kindav1beta1.DbInstanceSource{
					Generic: &kindav1beta1.GenericInstance{
						HostFrom:     &kindav1beta1.FromRef{Name: "host-sec"},
						PortFrom:     &kindav1beta1.FromRef{Name: "port-cm"},
						PublicIPFrom: &kindav1beta1.FromRef{Name: "ip-sec"},
					},
				},
			},
		}
		data := common.DbInstanceData{
			HostFrom:     &corev1.Secret{Data: map[string][]byte{"host": []byte("localhost")}},
			PortFrom:     &corev1.ConfigMap{Data: map[string]string{"port": "5432"}},
			PublicIPFrom: &corev1.Secret{Data: map[string][]byte{"ip": []byte("1.1.1.1")}},
		}

		checksums := common.GenerateDBInstanceChecksums(dbin, data)
		assert.Contains(t, checksums, "hostFrom")
		assert.Contains(t, checksums, "portFrom")
		assert.Contains(t, checksums, "publicIPFrom")
	})

	t.Run("Missing data doesn't panic and omits keys", func(t *testing.T) {
		// Test case: Spec expects references, but DbInstanceData is empty.
		// Verifies the function is robust against missing data (e.g. during partial reconcile).
		dbin := &kindav1beta1.DbInstance{
			Spec: kindav1beta1.DbInstanceSpec{
				DbInstanceSource: kindav1beta1.DbInstanceSource{
					Google: &kindav1beta1.GoogleInstance{},
				},
			},
		}
		data := common.DbInstanceData{}

		assert.NotPanics(t, func() {
			checksums := common.GenerateDBInstanceChecksums(dbin, data)
			assert.NotContains(t, checksums, "config")
			assert.NotContains(t, checksums, "adminSecret")
		})
	})

	t.Run("IsDBInstanceChanged detects modifications", func(t *testing.T) {
		// Test case: End-to-end detection of state change.
		// Verifies that modifying a referenced secret results in a 'changed' detection.
		dbin := &kindav1beta1.DbInstance{
			Status: kindav1beta1.DbInstanceStatus{
				Checksums: map[string]string{"spec": "initial-spec"},
			},
		}
		data := common.DbInstanceData{
			AdminSecret: &corev1.Secret{Data: map[string][]byte{"foo": []byte("bar")}},
		}

		// Initial state sync
		dbin.Status.Checksums = common.GenerateDBInstanceChecksums(dbin, data)
		assert.False(t, common.IsDBInstanceChanged(ctx, dbin, data))

		// Modify secret data
		data.AdminSecret.Data["foo"] = []byte("changed")
		assert.True(t, common.IsDBInstanceChanged(ctx, dbin, data))
	})
}

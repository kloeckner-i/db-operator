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
	"context"
	"errors"
	"testing"

	"bou.ke/monkey"
	kindav1beta11 "github.com/db-operator/db-operator/api/v1beta1"
	"github.com/db-operator/db-operator/pkg/test"
	"github.com/db-operator/db-operator/pkg/utils/kci"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	TestSecretName = "TestSec"
	TestNamespace  = "TestNS"
)

func newPostgresTestDbInstanceCr() kindav1beta11.DbInstance {
	info := make(map[string]string)
	info["DB_PORT"] = "5432"
	info["DB_CONN"] = "postgres"
	return kindav1beta11.DbInstance{
		Spec: kindav1beta11.DbInstanceSpec{
			Engine: "postgres",
			DbInstanceSource: kindav1beta11.DbInstanceSource{
				Generic: &kindav1beta11.GenericInstance{
					Host: test.GetPostgresHost(),
					Port: test.GetPostgresPort(),
				},
			},
		},
		Status: kindav1beta11.DbInstanceStatus{Info: info},
	}
}

func newPostgresTestDbCr(instanceRef kindav1beta11.DbInstance) *kindav1beta11.Database {
	o := metav1.ObjectMeta{Namespace: TestNamespace}
	s := kindav1beta11.DatabaseSpec{SecretName: TestSecretName}

	db := kindav1beta11.Database{
		ObjectMeta: o,
		Spec:       s,
		Status: kindav1beta11.DatabaseStatus{
			InstanceRef: &instanceRef,
		},
	}

	return &db
}

func newMysqlTestDbCr() *kindav1beta11.Database {
	o := metav1.ObjectMeta{Namespace: "TestNS"}
	s := kindav1beta11.DatabaseSpec{SecretName: "TestSec"}

	info := make(map[string]string)
	info["DB_PORT"] = "3306"
	info["DB_CONN"] = "mysql"

	db := kindav1beta11.Database{
		ObjectMeta: o,
		Spec:       s,
		Status: kindav1beta11.DatabaseStatus{
			InstanceRef: &kindav1beta11.DbInstance{
				Spec: kindav1beta11.DbInstanceSpec{
					Engine: "mysql",
					DbInstanceSource: kindav1beta11.DbInstanceSource{
						Generic: &kindav1beta11.GenericInstance{
							Host: test.GetMysqlHost(),
							Port: test.GetMysqlPort(),
						},
					},
				},
				Status: kindav1beta11.DbInstanceStatus{Info: info},
			},
		},
	}

	return &db
}

func TestIsSpecChanged(t *testing.T) {
	db := newPostgresTestDbCr(newPostgresTestDbInstanceCr())

	testDbSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: TestNamespace, Name: TestSecretName},
		Data: map[string][]byte{
			"POSTGRES_DB":       []byte("testdb"),
			"POSTGRES_USER":     []byte("testuser"),
			"POSTGRES_PASSWORD": []byte("testpassword"),
		},
	}

	addDBChecksum(db, testDbSecret)
	nochange := isDBChanged(db, testDbSecret)
	assert.Equal(t, nochange, false, "expected false")

	testDbSecret.Data["POSTGRES_PASSWORD"] = []byte("testpasswordNEW")

	change := isDBChanged(db, testDbSecret)
	assert.Equal(t, change, true, "expected true")
}

func testConfigmap1(_ context.Context, nsName types.NamespacedName) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	cm.Namespace = nsName.Namespace
	cm.Name = nsName.Name

	data := make(map[string]string)
	data["config"] = "test1"
	cm.Data = data

	return cm, nil
}

func testConfigmap2(_ context.Context, nsName types.NamespacedName) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	cm.Namespace = nsName.Namespace
	cm.Name = nsName.Name

	data := make(map[string]string)
	data["config"] = "test2"
	cm.Data = data

	return cm, nil
}

func errorConfigmap(namespace, configmapName string) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	return cm, errors.New("whatever error")
}

func testAdminSecret(namespace, secretName string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}

	data := make(map[string][]byte)
	data["user"] = []byte("user")
	data["password"] = []byte("securepassword")

	secret.Data = data
	return secret, nil
}

func TestSpecChanged(t *testing.T) {
	dbin := &kindav1beta11.DbInstance{}
	before := kindav1beta11.DbInstanceSpec{
		AdminUserSecret: kindav1beta11.NamespacedName{
			Namespace: "test",
			Name:      "secret1",
		},
	}

	ctx := context.Background()

	dbin.Spec = before
	addDBInstanceChecksumStatus(ctx, dbin)
	nochange := isDBInstanceSpecChanged(ctx, dbin)
	assert.Equal(t, nochange, false, "expected false")

	after := kindav1beta11.DbInstanceSpec{
		AdminUserSecret: kindav1beta11.NamespacedName{
			Namespace: "test",
			Name:      "secret2",
		},
	}
	dbin.Spec = after
	change := isDBInstanceSpecChanged(ctx, dbin)
	assert.Equal(t, change, true, "expected true")
}

func TestConfigChanged(t *testing.T) {
	dbin := &kindav1beta11.DbInstance{}
	dbin.Spec.Google = &kindav1beta11.GoogleInstance{
		InstanceName: "test",
		ConfigmapName: kindav1beta11.NamespacedName{
			Namespace: "testNS",
			Name:      "test",
		},
	}

	patch := monkey.Patch(kci.GetConfigResource, testConfigmap1)
	defer patch.Unpatch()
	addDBInstanceChecksumStatus(context.Background(), dbin)

	ctx := context.Background()

	nochange := isDBInstanceSpecChanged(ctx, dbin)
	assert.Equal(t, nochange, false, "expected false")

	patch = monkey.Patch(kci.GetConfigResource, testConfigmap2)
	change := isDBInstanceSpecChanged(ctx, dbin)
	assert.Equal(t, change, true, "expected true")
}

func TestAddChecksumStatus(t *testing.T) {
	dbin := &kindav1beta11.DbInstance{}
	addDBInstanceChecksumStatus(context.Background(), dbin)
	checksums := dbin.Status.Checksums
	assert.NotEqual(t, checksums, map[string]string{}, "annotation should have checksum")
}

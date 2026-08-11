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

package testutils

import (
	kindav1beta1 "github.com/db-operator/db-operator/v2/api/v1beta1"
	"github.com/db-operator/db-operator/v2/pkg/consts"
	"github.com/db-operator/db-operator/v2/pkg/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TestSecretName = "TestSec"
	TestNamespace  = "TestNS"
)

func NewPostgresTestDbInstanceCr() kindav1beta1.DbInstance {
	info := make(map[string]string)
	info["DB_PORT"] = "5432"
	info["DB_CONN"] = "postgres"
	return kindav1beta1.DbInstance{
		Spec: kindav1beta1.DbInstanceSpec{
			Engine: "postgres",
			DbInstanceSource: kindav1beta1.DbInstanceSource{
				Generic: &kindav1beta1.GenericInstance{
					Host: test.GetPostgresHost(),
					Port: test.GetPostgresPort(),
				},
			},
		},
		Status: kindav1beta1.DbInstanceStatus{Info: info},
	}
}

func NewMysqlTestDbInstanceCr() kindav1beta1.DbInstance {
	info := make(map[string]string)
	info["DB_PORT"] = "3306"
	info["DB_CONN"] = "mysql"
	return kindav1beta1.DbInstance{
		Spec: kindav1beta1.DbInstanceSpec{
			Engine: "mysql",
			DbInstanceSource: kindav1beta1.DbInstanceSource{
				Generic: &kindav1beta1.GenericInstance{
					Host: test.GetMysqlHost(),
					Port: test.GetMysqlPort(),
				},
			},
		},
		Status: kindav1beta1.DbInstanceStatus{Info: info},
	}
}

func NewPostgresTestDbCr(instanceRef kindav1beta1.DbInstance) *kindav1beta1.Database {
	o := metav1.ObjectMeta{Namespace: TestNamespace}
	s := kindav1beta1.DatabaseSpec{SecretName: TestSecretName}

	db := kindav1beta1.Database{
		ObjectMeta: o,
		Spec:       s,
		Status: kindav1beta1.DatabaseStatus{
			Engine: consts.ENGINE_POSTGRES,
		},
	}

	return &db
}

func NewMysqlTestDbCr() *kindav1beta1.Database {
	o := metav1.ObjectMeta{Namespace: "TestNS"}
	s := kindav1beta1.DatabaseSpec{SecretName: "TestSec"}

	info := make(map[string]string)
	info["DB_PORT"] = "3306"
	info["DB_CONN"] = "mysql"

	db := kindav1beta1.Database{
		ObjectMeta: o,
		Spec:       s,
		Status: kindav1beta1.DatabaseStatus{
			Engine: consts.ENGINE_MYSQL,
		},
	}

	return &db
}

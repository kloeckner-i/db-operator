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
	kciv1beta1 "github.com/kloeckner-i/db-operator/api/v1beta1"
	"github.com/kloeckner-i/db-operator/pkg/consts"
	"github.com/kloeckner-i/db-operator/pkg/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TestSecretName = "TestSec"
	TestNamespace  = "TestNS"
)

func NewPostgresTestDbInstanceCr() kciv1beta1.DbInstance {
	info := make(map[string]string)
	info["DB_PORT"] = "5432"
	info["DB_CONN"] = "postgres"
	return kciv1beta1.DbInstance{
		Spec: kciv1beta1.DbInstanceSpec{
			Engine: "postgres",
			DbInstanceSource: kciv1beta1.DbInstanceSource{
				Generic: &kciv1beta1.GenericInstance{
					Host: test.GetPostgresHost(),
					Port: test.GetPostgresPort(),
				},
			},
		},
		Status: kciv1beta1.DbInstanceStatus{Info: info},
	}
}

func NewMysqlTestDbInstanceCr() kciv1beta1.DbInstance {
	info := make(map[string]string)
	info["DB_PORT"] = "3306"
	info["DB_CONN"] = "mysql"
	return kciv1beta1.DbInstance{
		Spec: kciv1beta1.DbInstanceSpec{
			Engine: "mysql",
			DbInstanceSource: kciv1beta1.DbInstanceSource{
				Generic: &kciv1beta1.GenericInstance{
					Host: test.GetMysqlHost(),
					Port: test.GetMysqlPort(),
				},
			},
		},
		Status: kciv1beta1.DbInstanceStatus{Info: info},
	}
}

func NewPostgresTestDbCr(instanceRef kciv1beta1.DbInstance) *kciv1beta1.Database {
	o := metav1.ObjectMeta{Namespace: TestNamespace}
	s := kciv1beta1.DatabaseSpec{SecretName: TestSecretName}

	db := kciv1beta1.Database{
		ObjectMeta: o,
		Spec:       s,
		Status: kciv1beta1.DatabaseStatus{
			Engine: consts.ENGINE_POSTGRES,
		},
	}

	return &db
}

func NewMysqlTestDbCr() *kciv1beta1.Database {
	o := metav1.ObjectMeta{Namespace: "TestNS"}
	s := kciv1beta1.DatabaseSpec{SecretName: "TestSec"}

	info := make(map[string]string)
	info["DB_PORT"] = "3306"
	info["DB_CONN"] = "mysql"

	db := kciv1beta1.Database{
		ObjectMeta: o,
		Spec:       s,
		Status: kciv1beta1.DatabaseStatus{
			Engine: consts.ENGINE_MYSQL,
		},
	}

	return &db
}

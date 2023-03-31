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

package dbinstance

import "github.com/db-operator/db-operator/pkg/test"

func testGenericMysqlInstance() *Generic {
	return &Generic{
		Host:     test.GetMysqlHost(),
		Port:     test.GetMysqlPort(),
		Engine:   "mysql",
		User:     "root",
		Password: test.GetMysqlAdminPassword(),
	}
}

func testGenericPostgresInstance() *Generic {
	return &Generic{
		Host:     test.GetPostgresHost(),
		Port:     test.GetPostgresPort(),
		Engine:   "postgres",
		User:     "postgres",
		Password: test.GetPostgresAdminPassword(),
	}
}

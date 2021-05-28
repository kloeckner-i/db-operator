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

package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreatePostgres(t *testing.T) {
	p := testPostgres()
	p.Database = "testdb\""
	p.User = "testuser\""

	admin := getPostgresAdmin()

	err := Create(p, admin)
	assert.Errorf(t, err, "Should get error %v", err)

	p.Database = "testdb"
	err = Create(p, admin)
	assert.Errorf(t, err, "Should get error %v", err)

	p.User = "testuser"
	err = Create(p, admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)
}

func TestCreateMysql(t *testing.T) {
	m := testMysql()
	m.Database = "testdb\\'"
	m.User = "testuser\\'"

	admin := getMysqlAdmin()

	err := Create(m, admin)
	assert.Errorf(t, err, "Should get error %v", err)

	m.Database = "testdb"
	err = Create(m, admin)
	assert.Errorf(t, err, "Should get error %v", err)

	m.User = "testuser"
	err = Create(m, admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)
}

func TestDeletePostgres(t *testing.T) {
	p := testPostgres()
	admin := getPostgresAdmin()

	p.Database = "testdb"
	err := Delete(p, admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)
}

func TestDeleteMysql(t *testing.T) {
	m := testMysql()
	admin := getMysqlAdmin()

	m.Database = "testdb"
	err := Delete(m, admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)
}

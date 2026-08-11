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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreatePostgres(t *testing.T) {
	p, dbu := testPostgres()
	p.Database = "testdb\""
	dbu.Username = "testuser\""

	admin := getPostgresAdmin()

	err := CreateDatabase(context.TODO(), p, admin)
	assert.Errorf(t, err, "Should get error %v", err)

	p.Database = "testdb"
	err = CreateDatabase(context.TODO(), p, admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)

	err = CreateOrUpdateUser(context.TODO(), p, dbu, admin)
	assert.Errorf(t, err, "Should get error %v", err)

	dbu.Username = "testuser"
	err = CreateOrUpdateUser(context.TODO(), p, dbu, admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)
}

func TestCreateMysql(t *testing.T) {
	m, dbu := testMysql()
	dbu.Username = "testuser\\'"
	m.Database = "testdb\\'"

	admin := getMysqlAdmin()
	t.Log(m.Database)
	err := CreateDatabase(context.TODO(), m, admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)

	err = CreateUser(context.TODO(), m, dbu, admin)
	assert.Errorf(t, err, "Should get error %v", err)

	dbu.Username = "testuser"
	err = CreateUser(context.TODO(), m, dbu, admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)
}

func TestDeletePostgres(t *testing.T) {
	p, dbu := testPostgres()
	admin := getPostgresAdmin()

	p.Database = "testdb"
	err := DeleteDatabase(context.TODO(), p, admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)

	err = DeleteUser(context.TODO(), p, dbu, admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)
}

func TestDeleteMysql(t *testing.T) {
	m, dbu := testMysql()
	admin := getMysqlAdmin()

	m.Database = "testdb"
	err := DeleteDatabase(context.TODO(), m, admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)

	err = DeleteUser(context.TODO(), m, dbu, admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)
}

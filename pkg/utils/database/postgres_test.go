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

	"github.com/kloeckner-i/db-operator/pkg/test"
	"github.com/stretchr/testify/assert"
)

func testPostgres() *Postgres {
	return &Postgres{
		Backend:          "local",
		Host:             test.GetPostgresHost(),
		Port:             test.GetPostgresPort(),
		Database:         "testdb",
		User:             "testuser",
		Password:         "testpassword",
		Monitoring:       false,
		Extensions:       []string{},
		SSLEnabled:       false,
		SkipCAVerify:     false,
		DropPublicSchema: false,
		Schemas:          []string{},
	}
}

func getPostgresAdmin() AdminCredentials {
	return AdminCredentials{"postgres", test.GetPostgresAdminPassword()}
}

func TestPostgresExecuteQuery(t *testing.T) {
	testquery := "SELECT 1;"
	p := testPostgres()
	admin := getPostgresAdmin()

	assert.NoError(t, p.executeExec("postgres", testquery, admin))
}

func TestPostgresCreateDatabase(t *testing.T) {
	admin := getPostgresAdmin()
	p := testPostgres()

	err := p.createDatabase(admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)

	err = p.createDatabase(admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)
}

func TestPostgresCreateUser(t *testing.T) {
	admin := getPostgresAdmin()
	p := testPostgres()

	err := p.createUser(admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)

	err = p.createUser(admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)

	p.User = "testuser\""

	err = p.createUser(admin)
	assert.Error(t, err, "Should get error")
}

func TestPublicSchema(t *testing.T) {
	p := testPostgres()
	p.DropPublicSchema = false
	assert.NoError(t, p.checkSchemas())
}

func TestDropPublicSchemaFail(t *testing.T) {
	p := testPostgres()
	p.DropPublicSchema = true
	assert.Error(t, p.checkSchemas())
}

func TestDropPublicSchemaMonitoringTrue(t *testing.T) {
	p := testPostgres()
	admin := getPostgresAdmin()
	p.Monitoring = true
	p.DropPublicSchema = true
	p.dropPublicSchema(admin)
	assert.Error(t, p.checkSchemas())
}

func TestDropPublicSchemaMonitoringFalse(t *testing.T) {
	p := testPostgres()
	admin := getPostgresAdmin()
	p.Monitoring = false
	p.DropPublicSchema = true
	p.dropPublicSchema(admin)
	assert.NoError(t, p.checkSchemas())

	// Schemas is recreated here not to breaks tests
	p.Schemas = []string{"public"}
	assert.NoError(t, p.createSchemas(admin))
}

func TestEnableMonitoring(t *testing.T) {
	p := testPostgres()
	admin := getPostgresAdmin()
	p.Monitoring = true
	p.enableMonitoring(admin)
	p.Extensions = []string{"pg_stat_statements"}
	assert.NoError(t, p.checkExtensions())
}

func TestPostgresNoExtensions(t *testing.T) {
	admin := getPostgresAdmin()
	p := testPostgres()
	p.Extensions = []string{}

	assert.NoError(t, p.addExtensions(admin))
	assert.NoError(t, p.checkExtensions())
}

func TestPostgresAddExtensions(t *testing.T) {
	admin := getPostgresAdmin()
	p := testPostgres()
	p.Extensions = []string{"pgcrypto", "uuid-ossp"}

	assert.Error(t, p.checkExtensions())
	assert.NoError(t, p.addExtensions(admin))
	assert.NoError(t, p.checkExtensions())
}

func TestPostgresNoSchemas(t *testing.T) {
	admin := getPostgresAdmin()
	p := testPostgres()

	assert.NoError(t, p.checkSchemas())
	assert.NoError(t, p.createSchemas(admin))
	assert.NoError(t, p.checkSchemas())
}

func TestPostgresSchemas(t *testing.T) {
	admin := getPostgresAdmin()
	p := testPostgres()
	p.Schemas = []string{"schema_1", "schema_2"}

	assert.Error(t, p.checkSchemas())
	assert.NoError(t, p.createSchemas(admin))
	assert.NoError(t, p.checkSchemas())
}

func TestPostgresDeleteDatabase(t *testing.T) {
	admin := getPostgresAdmin()
	p := testPostgres()

	err := p.deleteDatabase(admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)

	err = p.deleteDatabase(admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)
}

func TestPostgresDeleteUser(t *testing.T) {
	admin := getPostgresAdmin()
	p := testPostgres()

	err := p.deleteUser(admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)
}

func TestPostgresGetCredentials(t *testing.T) {
	p := testPostgres()

	cred := p.GetCredentials()
	assert.Equal(t, cred.Username, p.User)
	assert.Equal(t, cred.Name, p.Database)
	assert.Equal(t, cred.Password, p.Password)
}

func TestPostgresParseAdminCredentials(t *testing.T) {
	p := testPostgres()

	invalidData := make(map[string][]byte)
	invalidData["unknownkey"] = []byte("wrong")

	_, err := p.ParseAdminCredentials(invalidData)
	assert.Errorf(t, err, "should get error %v", err)

	validData1 := make(map[string][]byte)
	validData1["user"] = []byte("admin")
	validData1["password"] = []byte("admin")

	cred, err := p.ParseAdminCredentials(validData1)
	assert.NoErrorf(t, err, "expected no error %v", err)
	assert.Equal(t, string(validData1["user"]), cred.Username, "expect same values")
	assert.Equal(t, string(validData1["password"]), cred.Password, "expect same values")

	validData2 := make(map[string][]byte)
	validData2["postgresql-password"] = []byte("passw0rd")
	cred, err = p.ParseAdminCredentials(validData2)
	assert.NoErrorf(t, err, "expected no error %v", err)
	assert.Equal(t, "postgres", cred.Username, "expect same values")
	assert.Equal(t, string(validData2["postgresql-password"]), cred.Password, "expect same values")

	validData3 := make(map[string][]byte)
	validData3["postgresql-postgres-password"] = []byte("passw0rd")
	cred, err = p.ParseAdminCredentials(validData3)
	assert.NoErrorf(t, err, "expected no error %v", err)
	assert.Equal(t, "postgres", cred.Username, "expect same values")
	assert.Equal(t, string(validData3["postgresql-postgres-password"]), cred.Password, "expect same values")
}

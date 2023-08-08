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
	"fmt"
	"testing"

	"github.com/db-operator/db-operator/pkg/test"
	"github.com/stretchr/testify/assert"
)

func testMysql() (*Mysql, *DatabaseUser) {
	return &Mysql{"local", test.GetMysqlHost(), test.GetMysqlPort(), "testdb", false, false}, &DatabaseUser{Username: "testuser", Password: "testpwd"}
}

func getMysqlAdmin() AdminCredentials {
	return AdminCredentials{"root", test.GetMysqlAdminPassword()}
}

func TestMysqlCheckStatus(t *testing.T) {
	m, dbu := testMysql()
	admin := getMysqlAdmin()
	assert.Error(t, m.CheckStatus(dbu))

	m.createOrUpdateUser(admin, dbu)
	assert.Error(t, m.CheckStatus(dbu))

	m.createDatabase(admin)
	assert.NoError(t, m.CheckStatus(dbu))

	m.deleteDatabase(admin)
	assert.Error(t, m.CheckStatus(dbu))

	m.deleteUser(admin, dbu)
	assert.Error(t, m.CheckStatus(dbu))

	m.Backend = "google"
	assert.Error(t, m.CheckStatus(dbu))
}

func TestMysqlExecuteQuery(t *testing.T) {
	testquery := "SELECT 1;"
	m, _ := testMysql()
	admin := getMysqlAdmin()
	assert.NoError(t, m.executeQuery(testquery, admin))

	admin.Password = "wrongpass"
	assert.Error(t, m.executeQuery(testquery, admin))
}

func TestMysqlCreateDatabase(t *testing.T) {
	admin := getMysqlAdmin()
	m, _ := testMysql()

	err := m.createDatabase(admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)

	err = m.createDatabase(admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)

	db, _ := m.getDbConn(admin.Username, admin.Password)
	defer db.Close()
	check := fmt.Sprintf("USE %s", m.Database)
	_, err = db.Exec(check)
	assert.NoError(t, err)
}

func TestMysqlCreateOrUpdateUser(t *testing.T) {
	admin := getMysqlAdmin()
	m, dbu := testMysql()

	err := m.createOrUpdateUser(admin, dbu)
	assert.NoError(t, err)

	err = m.createOrUpdateUser(admin, dbu)
	assert.NoError(t, err)

	assert.Equal(t, true, m.isUserExist(admin, dbu))
}

func TestMysqlDeleteDatabase(t *testing.T) {
	admin := getMysqlAdmin()
	m, _ := testMysql()

	err := m.deleteDatabase(admin)
	assert.NoError(t, err)

	err = m.deleteDatabase(admin)
	assert.NoError(t, err)

	db, _ := m.getDbConn(admin.Username, admin.Password)
	defer db.Close()
	check := fmt.Sprintf("USE %s", m.Database)
	_, err = db.Exec(check)
	assert.Error(t, err)
}

func TestMysqlDeleteUser(t *testing.T) {
	admin := getMysqlAdmin()
	m, dbu := testMysql()

	err := m.deleteUser(admin, dbu)
	assert.NoError(t, err)

	err = m.deleteUser(admin, dbu)
	assert.NoError(t, err)
	assert.Equal(t, false, m.isUserExist(admin, dbu))
}

func TestMysqlGetCredentials(t *testing.T) {
	m, dbu := testMysql()

	cred := m.GetCredentials(dbu)
	assert.Equal(t, cred.Username, dbu.Username)
	assert.Equal(t, cred.Name, m.Database)
	assert.Equal(t, cred.Password, dbu.Password)
}

func TestMysqlParseAdminCredentials(t *testing.T) {
	m, _:= testMysql()

	invalidData := make(map[string][]byte)
	invalidData["unknownkey"] = []byte("wrong")

	_, err := m.ParseAdminCredentials(invalidData)
	assert.Errorf(t, err, "should get error %v", err)

	validData1 := make(map[string][]byte)
	validData1["user"] = []byte("admin")
	validData1["password"] = []byte("admin")

	cred, err := m.ParseAdminCredentials(validData1)
	assert.NoErrorf(t, err, "expected no error %v", err)
	assert.Equal(t, string(validData1["user"]), cred.Username, "expect same values")
	assert.Equal(t, string(validData1["password"]), cred.Password, "expect same values")

	validData2 := make(map[string][]byte)
	validData2["mysql-root-password"] = []byte("passw0rd")
	cred, err = m.ParseAdminCredentials(validData2)
	assert.NoErrorf(t, err, "expected no error %v", err)
	assert.Equal(t, "root", cred.Username, "expect same values")
	assert.Equal(t, string(validData2["mysql-root-password"]), cred.Password, "expect same values")
}

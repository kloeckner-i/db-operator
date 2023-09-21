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
	return &Mysql{"local", test.GetMysqlHost(), test.GetMysqlPort(), "testdb", false, false}, &DatabaseUser{Username: "testuser", Password: "testpwd", AccessType: ACCESS_TYPE_MAINUSER}
}

func getMysqlAdmin() *DatabaseUser {
	return &DatabaseUser{
		Username: "root",
		Password: test.GetMysqlAdminPassword(),
	}
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
func TestMysqlMainUserLifecycle(t *testing.T) {
	// Test if it's created
	admin := getMysqlAdmin()
	m, dbu := testMysql()
	m.Database = "maintest"
	assert.NoError(t, m.createDatabase(admin))
	assert.NoError(t, m.setUserPermission(admin, dbu))

	createTable := `CREATE TABLE maintest.test_1 (
		role_id serial PRIMARY KEY,
		role_name VARCHAR (255) UNIQUE NOT NULL
	  );`
	assert.NoError(t, m.executeQueryAsUser(createTable, dbu))

	insert := "INSERT INTO maintest.test_1 VALUES (1, 'test-1')"
	assert.NoError(t, m.executeQueryAsUser(insert, dbu))

	selectQuery := "SELECT * FROM maintest.test_1"
	assert.NoError(t, m.executeQueryAsUser(selectQuery, dbu))

	insert = "INSERT INTO maintest.test_1 VALUES (2, 'test-2')"
	assert.NoError(t, m.executeQueryAsUser(insert, dbu))

	update := "UPDATE maintest.test_1 SET role_name = 'test-1-new' WHERE role_id = 1"
	assert.NoError(t, m.executeQueryAsUser(update, dbu))

	delete := "DELETE FROM maintest.test_1 WHERE role_id = 1"
	assert.NoError(t, m.executeQueryAsUser(delete, dbu))

	drop := "DROP TABLE maintest.test_1"
	assert.NoError(t, m.executeQueryAsUser(drop, dbu))
}

func TestMysqlReadOnlyUserLifecycle(t *testing.T) {
	// Test if it's created
	admin := getMysqlAdmin()
	m, dbu := testMysql()
	m.Database = "readonlytest"
	assert.NoError(t, m.createDatabase(admin))
	assert.NoError(t, m.setUserPermission(admin, dbu))
	readonlyUser := &DatabaseUser{
		Username:   "readonly",
		Password:   "123123",
		AccessType: ACCESS_TYPE_READONLY,
	}

	createTable := `CREATE TABLE readonlytest.test_1 (
		role_id serial PRIMARY KEY,
		role_name VARCHAR (255) UNIQUE NOT NULL
	  );`
	assert.NoError(t, m.executeQueryAsUser(createTable, dbu))

	err := m.createUser(admin, readonlyUser)
	assert.NoErrorf(t, err, "Unexpected error %v", err)

	// Test that it can't be created again
	err = m.createUser(admin, readonlyUser)
	assert.Error(t, err, "Was expecting an error")

	// Test that it can be updated
	err = m.updateUser(admin, readonlyUser)
	assert.NoErrorf(t, err, "Unexpected error %v", err)

	// Test that it has only readonly access to current objects
	createTable = `CREATE TABLE readonlytest.test_2 (
		role_id serial PRIMARY KEY,
		role_name VARCHAR (255) UNIQUE NOT NULL
	  );`
	assert.Error(t, m.executeQueryAsUser(createTable, readonlyUser))
	assert.NoError(t, m.executeQueryAsUser(createTable, dbu))

	insert := "INSERT INTO readonlytest.test_1 VALUES (1, 'test-1')"
	assert.NoError(t, m.executeQueryAsUser(insert, dbu))
	insert = "INSERT INTO readonlytest.test_2 VALUES (1, 'test-1')"
	assert.NoError(t, m.executeQueryAsUser(insert, dbu))

	selectQuery := "SELECT * FROM readonlytest.test_1"
	assert.NoError(t, m.executeQueryAsUser(selectQuery, readonlyUser))
	selectQuery = "SELECT * FROM readonlytest.test_2"
	assert.NoError(t, m.executeQueryAsUser(selectQuery, readonlyUser))

	insert = "INSERT INTO readonlytest.test_1 VALUES (2, 'test-2')"
	assert.Error(t, m.executeQueryAsUser(insert, readonlyUser))
	insert = "INSERT INTO readonlytest.test_2 VALUES (2, 'test-2')"
	assert.Error(t, m.executeQueryAsUser(insert, readonlyUser))

	update := "UPDATE readonlytest.test_1 SET role_name = 'test-1-new' WHERE role_id = 1"
	assert.Error(t, m.executeQueryAsUser(update, readonlyUser))
	update = "UPDATE readonlytest.test_2 SET role_name = 'test-1-new' WHERE role_id = 1"
	assert.Error(t, m.executeQueryAsUser(update, readonlyUser))

	delete := "DELETE FROM readonlytest.test_1 WHERE role_id = 1"
	assert.Error(t, m.executeQueryAsUser(delete, readonlyUser))
	delete = "DELETE FROM readonlytest.test_2 WHERE role_id = 1"
	assert.Error(t, m.executeQueryAsUser(delete, readonlyUser))

	drop := "DROP TABLE readonlytest.test_1"
	assert.Error(t, m.executeQueryAsUser(drop, readonlyUser))
	assert.NoError(t, m.executeQueryAsUser(drop, dbu))
	drop = "DROP TABLE readonlytest.test_2"
	assert.Error(t, m.executeQueryAsUser(drop, readonlyUser))
	assert.NoError(t, m.executeQueryAsUser(drop, dbu))

	// Test that it can be removed
	err = m.deleteUser(admin, readonlyUser)
	assert.NoErrorf(t, err, "Unexpected error %v", err)
}

func TestMysqlReadWriteUserLifecycle(t *testing.T) {
	// Test if it's created
	admin := getMysqlAdmin()
	m, dbu := testMysql()
	m.Database = "readwritetest"
	assert.NoError(t, m.createDatabase(admin))
	assert.NoError(t, m.setUserPermission(admin, dbu))
	readwriteUser := &DatabaseUser{
		Username:   "readwrite",
		Password:   "123123",
		AccessType: ACCESS_TYPE_READWRITE,
	}

	createTable := `CREATE TABLE readwritetest.test_1 (
		role_id serial PRIMARY KEY,
		role_name VARCHAR (255) UNIQUE NOT NULL
	  );`
	assert.NoError(t, m.executeQueryAsUser(createTable, dbu))

	err := m.createUser(admin, readwriteUser)
	assert.NoErrorf(t, err, "Unexpected error %v", err)

	// Test that it can't be created again
	err = m.createUser(admin, readwriteUser)
	assert.Error(t, err, "Was expecting an error")

	// Test that it can be updated
	err = m.updateUser(admin, readwriteUser)
	assert.NoErrorf(t, err, "Unexpected error %v", err)

	// Test that it has only readonly access to current objects
	createTable = `CREATE TABLE readwritetest.test_2 (
		role_id serial PRIMARY KEY,
		role_name VARCHAR (255) UNIQUE NOT NULL
	  );`
	assert.Error(t, m.executeQueryAsUser(createTable, readwriteUser))
	assert.NoError(t, m.executeQueryAsUser(createTable, dbu))

	insert := "INSERT INTO readwritetest.test_1 VALUES (1, 'test-1')"
	assert.NoError(t, m.executeQueryAsUser(insert, dbu))
	insert = "INSERT INTO readwritetest.test_2 VALUES (1, 'test-1')"
	assert.NoError(t, m.executeQueryAsUser(insert, dbu))
	insert = "INSERT INTO readwritetest.test_1 VALUES (2, 'test-2')"
	assert.NoError(t, m.executeQueryAsUser(insert, dbu))
	insert = "INSERT INTO readwritetest.test_2 VALUES (2, 'test-2')"
	assert.NoError(t, m.executeQueryAsUser(insert, dbu))

	selectQuery := "SELECT * FROM readwritetest.test_1"
	assert.NoError(t, m.executeQueryAsUser(selectQuery, readwriteUser))
	selectQuery = "SELECT * FROM readwritetest.test_2"
	assert.NoError(t, m.executeQueryAsUser(selectQuery, readwriteUser))

	insert = "INSERT INTO readwritetest.test_1 VALUES (3, 'test-3')"
	assert.NoError(t, m.executeQueryAsUser(insert, readwriteUser))
	insert = "INSERT INTO readwritetest.test_2 VALUES (3, 'test-3')"
	assert.NoError(t, m.executeQueryAsUser(insert, readwriteUser))

	update := "UPDATE readwritetest.test_1 SET role_name = 'test-1-new' WHERE role_id = 1"
	assert.NoError(t, m.executeQueryAsUser(update, readwriteUser))
	update = "UPDATE readwritetest.test_2 SET role_name = 'test-1-new' WHERE role_id = 1"
	assert.NoError(t, m.executeQueryAsUser(update, readwriteUser))

	delete := "DELETE FROM readwritetest.test_1 WHERE role_id = 2"
	assert.NoError(t, m.executeQueryAsUser(delete, readwriteUser))
	delete = "DELETE FROM readwritetest.test_2 WHERE role_id = 2"
	assert.NoError(t, m.executeQueryAsUser(delete, readwriteUser))

	drop := "DROP TABLE readwritetest.test_1"
	assert.Error(t, m.executeQueryAsUser(drop, readwriteUser))
	assert.NoError(t, m.executeQueryAsUser(drop, dbu))
	drop = "DROP TABLE readwritetest.test_2"
	assert.Error(t, m.executeQueryAsUser(drop, readwriteUser))
	assert.NoError(t, m.executeQueryAsUser(drop, dbu))

	// Test that it can be removed
	err = m.deleteUser(admin, readwriteUser)
	assert.NoErrorf(t, err, "Unexpected error %v", err)
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

func TestMysqlGetCredentials(t *testing.T) {
	m, dbu := testMysql()

	cred := m.GetCredentials(dbu)
	assert.Equal(t, cred.Username, dbu.Username)
	assert.Equal(t, cred.Name, m.Database)
	assert.Equal(t, cred.Password, dbu.Password)
}

func TestMysqlParseAdminCredentials(t *testing.T) {
	m, _ := testMysql()

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

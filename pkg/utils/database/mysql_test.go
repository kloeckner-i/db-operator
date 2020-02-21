package database

import (
	"fmt"
	"testing"

	"github.com/kloeckner-i/db-operator/pkg/test"

	"github.com/stretchr/testify/assert"
)

func testMysql() *Mysql {
	return &Mysql{"local", test.GetMysqlHost(), test.GetMysqlPort(), "testdb", "testuser", "testpwd"}
}

func getMysqlAdmin() AdminCredentials {
	return AdminCredentials{"root", test.GetMysqlAdminPassword()}
}

func TestMysqlCheckStatus(t *testing.T) {
	m := testMysql()
	admin := getMysqlAdmin()
	assert.Error(t, m.CheckStatus())

	m.createUser(admin)
	assert.Error(t, m.CheckStatus())

	m.createDatabase(admin)
	assert.NoError(t, m.CheckStatus())

	m.deleteDatabase(admin)
	assert.Error(t, m.CheckStatus())

	m.deleteUser(admin)
	assert.Error(t, m.CheckStatus())

	m.Backend = "google"
	assert.Error(t, m.CheckStatus())
}

func TestMysqlExecuteQuery(t *testing.T) {
	testquery := "SELECT 1;"
	m := testMysql()
	admin := getMysqlAdmin()
	assert.NoError(t, m.executeQuery(testquery, admin))

	admin.Password = "wrongpass"
	assert.Error(t, m.executeQuery(testquery, admin))
}

func TestMysqlCreateDatabase(t *testing.T) {
	admin := getMysqlAdmin()
	m := testMysql()

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

func TestMysqlCreateUser(t *testing.T) {
	admin := getMysqlAdmin()
	m := testMysql()

	err := m.createUser(admin)
	assert.NoError(t, err)

	err = m.createUser(admin)
	assert.NoError(t, err)

	assert.Equal(t, true, m.isUserExist(admin))
}

func TestMysqlDeleteDatabase(t *testing.T) {
	admin := getMysqlAdmin()
	m := testMysql()

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
	m := testMysql()

	err := m.deleteUser(admin)
	assert.NoError(t, err)

	err = m.deleteUser(admin)
	assert.NoError(t, err)
	assert.Equal(t, false, m.isUserExist(admin))
}

func TestMysqlGetCredentials(t *testing.T) {
	m := testMysql()

	cred := m.GetCredentials()
	assert.Equal(t, cred.Username, m.User)
	assert.Equal(t, cred.Name, m.Database)
	assert.Equal(t, cred.Password, m.Password)
}

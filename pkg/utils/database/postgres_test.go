package database

import (
	"testing"

	"github.com/kloeckner-i/db-operator/pkg/test"

	"github.com/stretchr/testify/assert"
)

func testPostgres() *Postgres {
	return &Postgres{"local", test.GetPostgresHost(), test.GetPostgresPort(), "testdb", "testuser", "testpassword", []string{}}
}

func getPostgresAdmin() AdminCredentials {
	return AdminCredentials{"postgres", test.GetPostgresAdminPassword()}
}

func TestPostgresExecuteQuery(t *testing.T) {
	testquery := "SELECT 1;"
	p := testPostgres()
	admin := getPostgresAdmin()

	assert.NoError(t, p.executeQuery("postgres", testquery, admin))
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

func TestPostgresNoExtensions(t *testing.T) {
	admin := getPostgresAdmin()
	p := testPostgres()

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

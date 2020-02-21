package database

import (
	database "github.com/kloeckner-i/db-operator/pkg/utils/database"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeterminPostgresType(t *testing.T) {
	postgresDbCr := newPostgresTestDbCr()
	dbcred := database.Credentials{Name: "testdb", Username: "testuser", Password: "password"}

	db, _ := determinDatabaseType(postgresDbCr, dbcred)
	_, ok := db.(database.Postgres)
	assert.Equal(t, ok, true, "expected true")
}

func TestDeterminMysqlType(t *testing.T) {
	mysqlDbCr := newMysqlTestDbCr()
	dbcred := database.Credentials{Name: "testdb", Username: "testuser", Password: "password"}

	db, _ := determinDatabaseType(mysqlDbCr, dbcred)
	_, ok := db.(database.Mysql)
	assert.Equal(t, ok, true, "expected true")
}

func TestParsePostgresSecretData(t *testing.T) {
	postgresDbCr := newPostgresTestDbCr()

	invalidData := make(map[string][]byte)
	invalidData["DB"] = []byte("testdb")

	_, err := parseDatabaseSecretData(postgresDbCr, invalidData)
	assert.Errorf(t, err, "should get error %v", err)

	validData := make(map[string][]byte)
	validData["POSTGRES_DB"] = []byte("testdb")
	validData["POSTGRES_USER"] = []byte("testuser")
	validData["POSTGRES_PASSWORD"] = []byte("testpassword")

	cred, err := parseDatabaseSecretData(postgresDbCr, validData)
	assert.NoErrorf(t, err, "expected no error %v", err)
	assert.Equal(t, string(validData["POSTGRES_DB"]), cred.Name, "expect same values")
	assert.Equal(t, string(validData["POSTGRES_USER"]), cred.Username, "expect same values")
	assert.Equal(t, string(validData["POSTGRES_PASSWORD"]), cred.Password, "expect same values")
}

func TestParseMysqlSecretData(t *testing.T) {
	mysqlDbCr := newMysqlTestDbCr()

	invalidData := make(map[string][]byte)
	invalidData["DB"] = []byte("testdb")

	_, err := parseDatabaseSecretData(mysqlDbCr, invalidData)
	assert.Errorf(t, err, "should get error %v", err)

	validData := make(map[string][]byte)
	validData["DB"] = []byte("testdb")
	validData["USER"] = []byte("testuser")
	validData["PASSWORD"] = []byte("testpassword")

	cred, err := parseDatabaseSecretData(mysqlDbCr, validData)
	assert.NoErrorf(t, err, "expected no error %v", err)
	assert.Equal(t, string(validData["DB"]), cred.Name, "expect same values")
	assert.Equal(t, string(validData["USER"]), cred.Username, "expect same values")
	assert.Equal(t, string(validData["PASSWORD"]), cred.Password, "expect same values")
}

func TestParseAdminSecretData(t *testing.T) {
	postgresDbCr := newPostgresTestDbCr()
	invalidData := make(map[string][]byte)
	invalidData["unknownkey"] = []byte("wrong")

	_, err := parseDatabaseAdminSecretData(postgresDbCr, invalidData)
	assert.Errorf(t, err, "should get error %v", err)

	validData := make(map[string][]byte)
	validData["user"] = []byte("admin")
	validData["password"] = []byte("admin")

	cred, err := parseDatabaseAdminSecretData(postgresDbCr, validData)
	assert.NoErrorf(t, err, "expected no error %v", err)
	assert.Equal(t, string(validData["user"]), cred.Username, "expect same values")
	assert.Equal(t, string(validData["password"]), cred.Password, "expect same values")
}

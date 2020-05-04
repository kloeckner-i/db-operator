package database

import (
	"testing"

	database "github.com/kloeckner-i/db-operator/pkg/utils/database"

	"github.com/stretchr/testify/assert"
)

var testDbcred = database.Credentials{Name: "testdb", Username: "testuser", Password: "password"}

func TestDeterminPostgresType(t *testing.T) {
	postgresDbCr := newPostgresTestDbCr(newPostgresTestDbInstanceCr())

	db, _ := determinDatabaseType(postgresDbCr, testDbcred)
	_, ok := db.(database.Postgres)
	assert.Equal(t, ok, true, "expected true")
}

func TestDeterminMysqlType(t *testing.T) {
	mysqlDbCr := newMysqlTestDbCr()

	db, _ := determinDatabaseType(mysqlDbCr, testDbcred)
	_, ok := db.(database.Mysql)
	assert.Equal(t, ok, true, "expected true")
}

func TestParsePostgresSecretData(t *testing.T) {
	postgresDbCr := newPostgresTestDbCr(newPostgresTestDbInstanceCr())

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

func TestMonitoringNotEnabled(t *testing.T) {
	instance := newPostgresTestDbInstanceCr()
	instance.Spec.Monitoring.Enabled = false
	postgresDbCr := newPostgresTestDbCr(instance)
	db, _ := determinDatabaseType(postgresDbCr, testDbcred)
	postgresInterface, _ := db.(database.Postgres)

	found := false
	for _, ext := range postgresInterface.Extensions {
		if ext == "pg_stat_statements" {
			found = true
			break
		}
	}
	assert.Equal(t, found, false, "expected pg_stat_statement is not included in extension list")
}

func TestMonitoringEnabled(t *testing.T) {
	instance := newPostgresTestDbInstanceCr()
	instance.Spec.Monitoring.Enabled = false
	postgresDbCr := newPostgresTestDbCr(instance)
	postgresDbCr.Status.InstanceRef.Spec.Monitoring.Enabled = true

	db, _ := determinDatabaseType(postgresDbCr, testDbcred)
	postgresInterface, _ := db.(database.Postgres)

	found := false
	for _, ext := range postgresInterface.Extensions {
		if ext == "pg_stat_statements" {
			found = true
			break
		}
	}
	assert.Equal(t, found, true, "expected pg_stat_statement is included in extension list")
}

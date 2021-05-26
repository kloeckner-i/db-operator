package dbinstance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMysqlGenericInstanceCreate if creating mysql generic instance works as expected
func TestMysqlGenericInstanceCreate(t *testing.T) {
	mysqlInstance := testGenericMysqlInstance()
	_, err := Create(mysqlInstance)
	assert.Error(t, err, "expected error already exits %v", err)

	mysqlInstance.Host = "wronghost"
	_, err = Create(mysqlInstance)
	assert.Error(t, err, "expected error %v", err)
}

// TestPostgresGenericInstanceCreate if creating postgres generic instance works as expected
func TestPostgresGenericInstanceCreate(t *testing.T) {
	postgresInstance := testGenericPostgresInstance()

	_, err := Create(postgresInstance)
	assert.Error(t, err, "expected error already exits %v", err)

	postgresInstance.Host = "wronghost"
	_, err = Create(postgresInstance)
	assert.Error(t, err, "expected no error %v", err)
}

// TestMysqlGenericInstanceUpdateNoError if upgrading mysql generic instance works as expected
func TestMysqlGenericInstanceUpdateNoError(t *testing.T) {
	mysqlInstance := testGenericMysqlInstance()
	_, err := Update(mysqlInstance)
	assert.NoError(t, err, "expected no error %v", err)
}

// TestMysqlGenericInstanceUpdateNonExist checks upgrading non existing mysql generic instance throws error
func TestMysqlGenericInstanceUpdateNonExist(t *testing.T) {
	mysqlInstance := testGenericMysqlInstance()
	mysqlInstance.Host = "wronghost"
	_, err := Update(mysqlInstance)
	assert.Error(t, err, "expected error %v", err)
}

// TestPostgresGenericInstanceUpdate if upgrading postgres generic instance works as expected
func TestPostgresGenericInstanceUpdate(t *testing.T) {
	postgresInstance := testGenericPostgresInstance()
	_, err := Update(postgresInstance)
	assert.NoError(t, err, "expected no error %v", err)

	postgresInstance.Host = "wronghost"
	_, err = Update(postgresInstance)
	assert.Error(t, err, "expected error %v", err)
}

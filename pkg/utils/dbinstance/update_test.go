package dbinstance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

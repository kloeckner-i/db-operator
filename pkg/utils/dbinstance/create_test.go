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

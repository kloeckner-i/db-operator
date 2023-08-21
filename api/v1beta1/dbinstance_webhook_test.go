package v1beta1_test

import (
	"testing"

	"github.com/db-operator/db-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
)

func TestUnitEngineValid(t *testing.T) {
	err := v1beta1.ValidateEngine("postgres")
	assert.NoError(t, err)

	err = v1beta1.ValidateEngine("mysql")
	assert.NoError(t, err)
}
func TestUnitEngineInvalid(t *testing.T) {
	err := v1beta1.ValidateEngine("dummy")
	assert.Error(t, err)
}

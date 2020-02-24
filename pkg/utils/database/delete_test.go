package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeletePostgres(t *testing.T) {
	p := testPostgres()
	admin := getPostgresAdmin()

	p.Database = "testdb"
	err := Delete(p, admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)
}

func TestDeleteMysql(t *testing.T) {
	m := testMysql()
	admin := getMysqlAdmin()

	m.Database = "testdb"
	err := Delete(m, admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)
}

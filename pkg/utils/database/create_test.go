package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreatePostgres(t *testing.T) {
	p := testPostgres()
	p.Database = "testdb\""
	p.User = "testuser\""

	admin := getPostgresAdmin()

	err := Create(p, admin)
	assert.Errorf(t, err, "Should get error %v", err)

	p.Database = "testdb"
	err = Create(p, admin)
	assert.Errorf(t, err, "Should get error %v", err)

	p.User = "testuser"
	err = Create(p, admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)
}

func TestCreateMysql(t *testing.T) {
	m := testMysql()
	m.Database = "testdb\\'"
	m.User = "testuser\\'"

	admin := getMysqlAdmin()

	err := Create(m, admin)
	assert.Errorf(t, err, "Should get error %v", err)

	m.Database = "testdb"
	err = Create(m, admin)
	assert.Errorf(t, err, "Should get error %v", err)

	m.User = "testuser"
	err = Create(m, admin)
	assert.NoErrorf(t, err, "Unexpected error %v", err)
}

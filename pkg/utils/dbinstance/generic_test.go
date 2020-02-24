package dbinstance

import "github.com/kloeckner-i/db-operator/pkg/test"

func testGenericMysqlInstance() *Generic {
	return &Generic{
		Host:     test.GetMysqlHost(),
		Port:     test.GetMysqlPort(),
		Engine:   "mysql",
		User:     "root",
		Password: test.GetMysqlAdminPassword(),
	}
}

func testGenericPostgresInstance() *Generic {
	return &Generic{
		Host:     test.GetPostgresHost(),
		Port:     test.GetPostgresPort(),
		Engine:   "postgres",
		User:     "postgres",
		Password: test.GetPostgresAdminPassword(),
	}
}

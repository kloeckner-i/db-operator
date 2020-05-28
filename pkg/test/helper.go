package test

import (
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
)

// GetMysqlHost set mysql host which used by unit test
func GetMysqlHost() string {
	if value, ok := os.LookupEnv("MYSQL_HOST"); ok {
		return value
	}
	return "127.0.0.1"
}

// GetMysqlPort set mysql port which used by unit test
func GetMysqlPort() uint16 {
	if value, ok := os.LookupEnv("MYSQL_PORT"); ok {
		port, err := strconv.Atoi(value)
		if err != nil {
			logrus.Fatal(err)
		}
		return uint16(port)
	}
	return 3306
}

// GetMysqlAdminPassword set mysql password which used by unit test
func GetMysqlAdminPassword() string {
	if value, ok := os.LookupEnv("MYSQL_PASSWORD"); ok {
		return value
	}
	return "test1234"
}

// GetPostgresHost set postgres host which used by unit test
func GetPostgresHost() string {
	if value, ok := os.LookupEnv("POSTGRES_HOST"); ok {
		return value
	}
	return "127.0.0.1"
}

// GetPostgresPort set postgres port which used by unit test
func GetPostgresPort() uint16 {
	if value, ok := os.LookupEnv("POSTGRES_PORT"); ok {
		port, err := strconv.Atoi(value)
		if err != nil {
			logrus.Fatal(err)
		}
		return uint16(port)
	}
	return 5432
}

// GetPostgresAdminPassword set postgres password which used by unit test
func GetPostgresAdminPassword() string {
	if value, ok := os.LookupEnv("POSTGRES_PASSWORD"); ok {
		return value
	}
	return "test1234"
}

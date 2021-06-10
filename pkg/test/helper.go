/*
 * Copyright 2021 kloeckner.i GmbH
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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

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

package database

const (
	ACCESS_TYPE_READONLY  = "readOnly"
	ACCESS_TYPE_READWRITE = "readWrite"
	ACCESS_TYPE_MAINUSER  = "main"
)

// Credentials contains credentials to connect database
type Credentials struct {
	Name             string
	Username         string
	Password         string
	TemplatedSecrets map[string]string
}

type DatabaseUser struct {
	Username   string
	Password   string
	AccessType string
}

func (user *DatabaseUser) SetAccessType(accessType string) {
	user.AccessType = accessType
}

// DatabaseAddress contains host and port of a database instance
type DatabaseAddress struct {
	Host string
	Port uint16
}

// AdminCredentials contains admin username and password of database server
type AdminCredentials struct {
	Username string `yaml:"user"`
	Password string `yaml:"password"`
}

// Database is interface for CRUD operate of different types of databases
type Database interface {
	CheckStatus(user *DatabaseUser) error
	GetCredentials(user *DatabaseUser) Credentials
	ParseAdminCredentials(data map[string][]byte) (AdminCredentials, error)
	GetDatabaseAddress() DatabaseAddress
	createDatabase(admin AdminCredentials) error
	deleteDatabase(admin AdminCredentials) error
	createOrUpdateUser(admin AdminCredentials, user *DatabaseUser) error
	createUser(admin AdminCredentials, user *DatabaseUser) error
	updateUser(admin AdminCredentials, user *DatabaseUser) error
	deleteUser(admin AdminCredentials, user *DatabaseUser) error
	setUserPermission(admin AdminCredentials, user *DatabaseUser) error
}

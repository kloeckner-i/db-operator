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

import "context"

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
	Username        string `yaml:"user"`
	Password        string `yaml:"password"`
	AccessType      string
	ExtraPrivileges []string
	GrantToAdmin    bool
	// A workaround mostly for AWS RDS. Since we can't
	// grant a user that is a member of rds_iam role to
	// the admin, we need to make the grant after a
	// user is revoked from rds_iam, hence it should
	// happen while it's being deleted
	GrantToAdminOnDelete bool
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
	CheckStatus(ctx context.Context, user *DatabaseUser) error
	GetCredentials(ctx context.Context, user *DatabaseUser) Credentials
	ParseAdminCredentials(ctx context.Context, data map[string][]byte) (*DatabaseUser, error)
	GetDatabaseAddress(ctx context.Context) DatabaseAddress
	QueryAsUser(ctx context.Context, query string, user *DatabaseUser) (string, error)
	createDatabase(ctx context.Context, admin *DatabaseUser) error
	deleteDatabase(ctx context.Context, admin *DatabaseUser) error
	createOrUpdateUser(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error
	createUser(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error
	updateUser(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error
	deleteUser(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error
	setUserPermission(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error
	revokePermissions(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error
	execAsUser(ctx context.Context, query string, user *DatabaseUser) error
}

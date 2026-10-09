/*
 * Copyright 2023 DB-Operator Authors
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
	DB_DUMMY_HOSTNAME     = "hostname"
	DB_DUMMY_PORT         = 1122
	DBUSER_DUMMY_USERNAME = "username"
	DBUSER_DUMMY_PASSWORD = "qwertyu9"
)

func NewDummyUser(access string) *DatabaseUser {
	return &DatabaseUser{
		Username:   DBUSER_DUMMY_USERNAME,
		Password:   DBUSER_DUMMY_PASSWORD,
		AccessType: access,
	}
}

// Dummy is a database interface implementation for unit testing
type Dummy struct {
	Error error
}

// QueryAsUser implements Database.
func (d Dummy) QueryAsUser(ctx context.Context, query string, user *DatabaseUser) (string, error) {
	if d.Error != nil {
		return "", d.Error
	}
	return query, nil
}

// CheckStatus implements Database.
func (Dummy) CheckStatus(ctx context.Context, user *DatabaseUser) error {
	panic("unimplemented")
}

// GetCredentials implements Database.
func (Dummy) GetCredentials(ctx context.Context, user *DatabaseUser) Credentials {
	panic("unimplemented")
}

// GetDatabaseAddress implements Database.
func (Dummy) GetDatabaseAddress(ctx context.Context) DatabaseAddress {
	return DatabaseAddress{
		Host: DB_DUMMY_HOSTNAME,
		Port: DB_DUMMY_PORT,
	}
}

// ParseAdminCredentials implements Database.
func (Dummy) ParseAdminCredentials(ctx context.Context, data map[string][]byte) (*DatabaseUser, error) {
	panic("unimplemented")
}

// createDatabase implements Database.
func (Dummy) createDatabase(ctx context.Context, admin *DatabaseUser) error {
	panic("unimplemented")
}

// createOrUpdateUser implements Database.
func (Dummy) createOrUpdateUser(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error {
	panic("unimplemented")
}

// createUser implements Database.
func (Dummy) createUser(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error {
	panic("unimplemented")
}

// deleteDatabase implements Database.
func (Dummy) deleteDatabase(ctx context.Context, admin *DatabaseUser) error {
	panic("unimplemented")
}

// deleteUser implements Database.
func (Dummy) deleteUser(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error {
	panic("unimplemented")
}

// execAsUser implements Database.
func (Dummy) execAsUser(ctx context.Context, query string, user *DatabaseUser) error {
	panic("unimplemented")
}

// setUserPermission implements Database.
func (Dummy) setUserPermission(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error {
	panic("unimplemented")
}

func (p Dummy) revokePermissions(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error {
	panic("unimplemented")
}

// updateUser implements Database.
func (Dummy) updateUser(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error {
	panic("unimplemented")
}

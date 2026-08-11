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

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/mysql"
	"sigs.k8s.io/controller-runtime/pkg/log"

	// do not delete
	"github.com/db-operator/db-operator/v2/pkg/utils/kci"
	_ "github.com/go-sql-driver/mysql"
)

// Mysql is a database interface, abstracted object
// represents a database on mysql instance
// can be used to execute query to mysql database
type Mysql struct {
	Backend      string
	Host         string
	Port         uint16
	Database     string
	SSLEnabled   bool
	SkipCAVerify bool
}

const mysqlDefaultSSLMode = "preferred"

// Internal helpers, these functions are not part for the `Database` interface

func (m Mysql) sslMode() string {
	if !m.SSLEnabled {
		return "false"
	}

	if m.SSLEnabled && !m.SkipCAVerify {
		return "true"
	}

	if m.SSLEnabled && m.SkipCAVerify {
		return "skip-verify"
	}

	return mysqlDefaultSSLMode
}

func (m Mysql) getDbConn(ctx context.Context, user, password string) (*sql.DB, error) {
	log := log.FromContext(ctx)
	var db *sql.DB
	var err error

	switch m.Backend {
	case "google":
		// TODO: DialPassword is deprecated, it should be gone
		db, err = mysql.DialPassword(m.Host, user, password) //nolint:all
		if err != nil {
			log.Error(err, "failed to validate db connection")
			return db, err
		}
	default:
		dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%d)/?tls=%s", user, password, m.Host, m.Port, m.sslMode())
		db, err = sql.Open("mysql", dataSourceName)
		if err != nil {
			log.Error(err, "failed to validate db connection")
			return db, err
		}
		db.SetMaxIdleConns(0)
	}

	return db, nil
}

func (m Mysql) executeQuery(ctx context.Context, query string, admin *DatabaseUser) error {
	log := log.FromContext(ctx)
	db, err := m.getDbConn(ctx, admin.Username, admin.Password)
	if err != nil {
		log.Error(err, "failed to get db connection")
		return err
	}
	defer db.Close()

	rows, err := db.Query(query)
	if err != nil {
		log.Error(err, "failed to get db connection")
		return err
	}
	rows.Close()

	return nil
}

func (m Mysql) execAsUser(ctx context.Context, query string, user *DatabaseUser) error {
	log := log.FromContext(ctx)
	db, err := m.getDbConn(ctx, user.Username, user.Password)
	if err != nil {
		log.Error(err, "failed to get db connection")
		return err
	}
	defer db.Close()

	rows, err := db.Query(query)
	if err != nil {
		log.Error(err, "failed to get db connection")
		return err
	}
	rows.Close()

	return nil
}

func (m Mysql) isRowExist(ctx context.Context, query string, admin *DatabaseUser) bool {
	log := log.FromContext(ctx)
	db, err := m.getDbConn(ctx, admin.Username, admin.Password)
	if err != nil {
		log.Error(err, "failed to get db connection")
		return false
	}
	defer db.Close()

	var result string
	err = db.QueryRow(query).Scan(&result)
	if err != nil {
		log.Error(err, "failed to get db connection")
		return false
	}

	return true
}

func (m Mysql) isUserExist(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) bool {
	check := fmt.Sprintf("SELECT User FROM mysql.user WHERE user='%s';", user.Username)
	return m.isRowExist(ctx, check, admin)
}

// Functions that implement the `Database` interface

// CheckStatus checks status of mysql database
// if the connection to database works
func (m Mysql) CheckStatus(ctx context.Context, user *DatabaseUser) error {
	db, err := m.getDbConn(ctx, user.Username, user.Password)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("db conn test failed - could not establish a connection: %v", err)
	}

	check := fmt.Sprintf("USE %s", m.Database)
	if _, err := db.Exec(check); err != nil {
		return err
	}

	return nil
}

// GetCredentials returns credentials of the mysql database
func (m Mysql) GetCredentials(ctx context.Context, user *DatabaseUser) Credentials {
	return Credentials{
		Name:     m.Database,
		Username: user.Username,
		Password: user.Password,
	}
}

// ParseAdminCredentials parse admin username and password of mysql database from secret data
// If "user" key is not defined, take "root" as admin user by default
func (m Mysql) ParseAdminCredentials(ctx context.Context, data map[string][]byte) (*DatabaseUser, error) {
	admin := &DatabaseUser{}

	_, ok := data["user"]
	if ok {
		admin.Username = string(data["user"])
	} else {
		// default admin username is "root"
		admin.Username = "root"
	}

	// if "password" key is defined in data, take value as password
	_, ok = data["password"]
	if ok {
		admin.Password = string(data["password"])
		return admin, nil
	}

	// take value of "mysql-root-password" key as password if "password" key is not defined in data
	// it's compatible with secret created by stable mysql chart
	_, ok = data["mysql-root-password"]
	if ok {
		admin.Password = string(data["mysql-root-password"])
		return admin, nil
	}

	return admin, errors.New("can not find mysql admin credentials")
}

func (m Mysql) GetDatabaseAddress(ctx context.Context) DatabaseAddress {
	return DatabaseAddress{
		Host: m.Host,
		Port: m.Port,
	}
}

func (m Mysql) QueryAsUser(ctx context.Context, query string, user *DatabaseUser) (string, error) {
	log := log.FromContext(ctx)
	db, err := m.getDbConn(ctx, user.Username, user.Password)
	if err != nil {
		log.Error(err, "failed to get db connection")
		return "", err
	}
	defer db.Close()

	var result string
	if err := db.QueryRow(query).Scan(&result); err != nil {
		log.Error(err, "an error occurred while executing a query")
		return "", err
	}
	return result, nil
}

func (m Mysql) createDatabase(ctx context.Context, admin *DatabaseUser) error {
	create := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`;", m.Database)

	err := m.executeQuery(ctx, create, admin)
	if err != nil {
		return err
	}

	return nil
}

func (m Mysql) deleteDatabase(ctx context.Context, admin *DatabaseUser) error {
	log := log.FromContext(ctx)
	create := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", m.Database)

	err := kci.Retry(3, 5*time.Second, func() error {
		err := m.executeQuery(ctx, create, admin)
		if err != nil {
			log.V(2).Info("failed with error, retrying", "error", err)
			return err
		}

		return nil
	})
	if err != nil {
		log.V(2).Info("failed with error, retrying", "error", err)
		return err
	}

	return nil
}

func (m Mysql) createOrUpdateUser(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error {
	if !m.isUserExist(ctx, admin, user) {
		if err := m.createUser(ctx, admin, user); err != nil {
			return err
		}
	} else {
		if err := m.updateUser(ctx, admin, user); err != nil {
			return err
		}
	}

	if err := m.setUserPermission(ctx, admin, user); err != nil {
		return err
	}

	return nil
}

func (m Mysql) createUser(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error {
	create := fmt.Sprintf("CREATE USER `%s` IDENTIFIED BY '%s';", user.Username, user.Password)

	if !m.isUserExist(ctx, admin, user) {
		err := m.executeQuery(ctx, create, admin)
		if err != nil {
			return err
		}
	} else {
		err := fmt.Errorf("user already exists: %s", user.Username)
		return err
	}

	if err := m.setUserPermission(ctx, admin, user); err != nil {
		return err
	}

	return nil
}

func (m Mysql) updateUser(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error {
	update := fmt.Sprintf("ALTER USER `%s` IDENTIFIED BY '%s';", user.Username, user.Password)

	if !m.isUserExist(ctx, admin, user) {
		err := fmt.Errorf("user doesn't exist yet: %s", user.Username)
		return err
	} else {
		err := m.executeQuery(ctx, update, admin)
		if err != nil {
			return err
		}
	}

	if err := m.setUserPermission(ctx, admin, user); err != nil {
		return err
	}

	return nil
}

func (m Mysql) setUserPermission(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error {
	switch user.AccessType {
	case ACCESS_TYPE_MAINUSER:
		grant := fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%';", m.Database, user.Username)
		err := m.executeQuery(ctx, grant, admin)
		if err != nil {
			return err
		}
	case ACCESS_TYPE_READONLY:
		grant := fmt.Sprintf("GRANT SELECT ON `%s`.* TO '%s'@'%%';", m.Database, user.Username)
		err := m.executeQuery(ctx, grant, admin)
		if err != nil {
			return err
		}
	case ACCESS_TYPE_READWRITE:
		grant := fmt.Sprintf("GRANT SELECT, UPDATE, INSERT, DELETE ON `%s`.* TO '%s'@'%%';", m.Database, user.Username)
		err := m.executeQuery(ctx, grant, admin)
		if err != nil {
			return err
		}
	default:
		err := fmt.Errorf("unknown access type: %s", user.AccessType)
		return err
	}
	for _, role := range user.ExtraPrivileges {
		grantRole := fmt.Sprintf("GRANT \"%s\" to \"%s\"", role, user.Username)
		if err := m.executeQuery(ctx, grantRole, admin); err != nil {
			return err
		}
	}

	return nil
}

// Not implemented on mysql, but required for the interface
func (p Mysql) revokePermissions(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error {
	log.FromContext(ctx).
		Info("Permissions can't be revoked on Mysql instance, please do it manually")
	return nil
}

func (m Mysql) deleteUser(ctx context.Context, admin *DatabaseUser, user *DatabaseUser) error {
	delete := fmt.Sprintf("DROP USER `%s`;", user.Username)

	if m.isUserExist(ctx, admin, user) {
		err := m.executeQuery(ctx, delete, admin)
		if err != nil {
			return err
		}
	}

	return nil
}

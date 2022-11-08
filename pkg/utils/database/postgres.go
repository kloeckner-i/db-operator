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
	"database/sql"
	"errors"
	"fmt"
	"time"

	// Don't delete below package. Used for driver "cloudsqlpostgres"
	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/postgres"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"

	// Don't delete below package. Used for driver "postgres"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Postgres is a database interface, abstraced object
// represents a database on postgres instance
// can be used to execute query to postgres database
type Postgres struct {
	Backend          string
	Host             string
	Port             uint16
	Database         string
	User             string
	Password         string
	Monitoring       bool
	Extensions       []string
	SSLEnabled       bool
	SkipCAVerify     bool
	DropPublicSchema bool
	Schemas          []string
}

const postgresDefaultSSLMode = "disable"

func (p Postgres) sslMode() string {
	if !p.SSLEnabled {
		return "disable"
	}

	if p.SSLEnabled && !p.SkipCAVerify {
		return "verify-ca"
	}

	if p.SSLEnabled && p.SkipCAVerify {
		return "require"
	}

	return postgresDefaultSSLMode
}

func (p Postgres) getDbConn(dbname, user, password string) (*sql.DB, error) {
	var db *sql.DB
	var sqldriver string

	switch p.Backend {
	case "google":
		sqldriver = "cloudsqlpostgres"
	default:
		sqldriver = "postgres"
	}

	dataSourceName := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s", p.Host, p.Port, dbname, user, password, p.sslMode())
	db, err := sql.Open(sqldriver, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %v", err)
	}

	return db, err
}

func (p Postgres) executeExec(database, query string, admin AdminCredentials) error {
	db, err := p.getDbConn(database, admin.Username, admin.Password)
	if err != nil {
		logrus.Fatalf("failed to open db connection: %s", err)
	}

	defer db.Close()
	_, err = db.Exec(query)

	return err
}

func (p Postgres) isDbExist(admin AdminCredentials) bool {
	check := fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname = '%s';", p.Database)

	return p.isRowExist("postgres", check, admin.Username, admin.Password)
}

func (p Postgres) isUserExist(admin AdminCredentials) bool {
	check := fmt.Sprintf("SELECT 1 FROM pg_user WHERE usename = '%s';", p.User)

	return p.isRowExist("postgres", check, admin.Username, admin.Password)
}

// CheckStatus checks status of postgres database
// if the connection to database works
func (p Postgres) CheckStatus() error {
	db, err := p.getDbConn(p.Database, p.User, p.Password)
	if err != nil {
		return fmt.Errorf("db conn test failed - couldn't get db conn: %s", err)
	}
	defer db.Close()
	_, err = db.Query("SELECT 1")
	if err != nil {
		return fmt.Errorf("db conn test failed - failed to execute query: %s", err)
	}

	if err := p.checkSchemas(); err != nil {
		return err
	}

	if err := p.checkExtensions(); err != nil {
		return err
	}

	return nil
}

func (p Postgres) isRowExist(database, query, user, password string) bool {
	db, err := p.getDbConn(database, user, password)
	if err != nil {
		logrus.Fatal(err)
	}
	defer db.Close()

	var name string
	err = db.QueryRow(query).Scan(&name)
	if err != nil {
		logrus.Debugf("failed executing query %s - %s", query, err)
		return false
	}
	return true
}

func (p Postgres) createDatabase(admin AdminCredentials) error {
	create := fmt.Sprintf("CREATE DATABASE \"%s\";", p.Database)

	if !p.isDbExist(admin) {
		err := p.executeExec("postgres", create, admin)
		if err != nil {
			logrus.Errorf("failed creating postgres database %s", err)
			return err
		}
	}

	if p.Monitoring {
		err := p.enableMonitoring(admin)
		if err != nil {
			return fmt.Errorf("can not enable monitoring - %s", err)
		}
	}

	err := p.addExtensions(admin)
	if err != nil {
		return fmt.Errorf("can not add extension - %s", err)
	}

	if p.DropPublicSchema {
		if err := p.dropPublicSchema(admin); err != nil {
			return fmt.Errorf("can not drop public schema - %s", err)
		}
		if len(p.Schemas) == 0 {
			logrus.Info("the public schema is dropped, but no additional schemas are created, schema creation must be handled on the application side now")
		}
	}

	if len(p.Schemas) > 0 {
		if err := p.createSchemas(admin); err != nil {
			logrus.Errorf("failed creating additional schemas %s", err)
			return err
		}
	}

	return nil
}

func (p Postgres) createUser(admin AdminCredentials) error {
	create := fmt.Sprintf("CREATE USER \"%s\" WITH ENCRYPTED PASSWORD '%s' NOSUPERUSER;", p.User, p.Password)
	grant := fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE \"%s\" TO \"%s\";", p.Database, p.User)
	update := fmt.Sprintf("ALTER ROLE \"%s\" WITH ENCRYPTED PASSWORD '%s';", p.User, p.Password)

	if !p.isUserExist(admin) {
		err := p.executeExec("postgres", create, admin)
		if err != nil {
			logrus.Errorf("failed creating postgres user - %s", err)
			return err
		}
	} else {
		err := p.executeExec("postgres", update, admin)
		if err != nil {
			logrus.Errorf("failed updating postgres user %s - %s", update, err)
			return err
		}
	}

	err := p.executeExec("postgres", grant, admin)
	if err != nil {
		logrus.Errorf("failed granting postgres user %s - %s", grant, err)
		return err
	}

	for _, s := range p.Schemas {
		grantUserAccess := fmt.Sprintf("GRANT ALL ON SCHEMA \"%s\" TO \"%s\"", s, p.User)
		if err := p.executeExec(p.Database, grantUserAccess, admin); err != nil {
			logrus.Errorf("failed to grant usage access to %s on schema %s: %s", p.User, s, err)
			return err
		}
	}
	return nil
}

func (p Postgres) dropPublicSchema(admin AdminCredentials) error {
	if p.Monitoring {
		return fmt.Errorf("can not drop public schema when monitoring is enabled on instance level")
	}

	drop := "DROP SCHEMA IF EXISTS public;"
	if err := p.executeExec(p.Database, drop, admin); err != nil {
		logrus.Errorf("failed to drop the schema Public: %s", err)
		return err
	}
	return nil
}

func (p Postgres) createSchemas(admin AdminCredentials) error {
	for _, s := range p.Schemas {
		createSchema := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", s)
		if err := p.executeExec(p.Database, createSchema, admin); err != nil {
			logrus.Errorf("failed to create schema %s, %s", s, err)
			return err
		}
	}

	return nil
}

func (p Postgres) checkSchemas() error {
	if p.DropPublicSchema {
		query := "SELECT 1 FROM pg_catalog.pg_namespace WHERE nspname = 'public';"
		if p.isRowExist(p.Database, query, p.User, p.Password) {
			return fmt.Errorf("schema public still exists")
		}
	}
	for _, s := range p.Schemas {
		query := fmt.Sprintf("SELECT 1 FROM pg_catalog.pg_namespace WHERE nspname = '%s';", s)
		if !p.isRowExist(p.Database, query, p.User, p.Password) {
			return fmt.Errorf("couldn't find schema %s in database %s", s, p.Database)
		}
	}
	return nil
}

func (p Postgres) deleteDatabase(admin AdminCredentials) error {
	revoke := fmt.Sprintf("REVOKE CONNECT ON DATABASE \"%s\" FROM PUBLIC, \"%s\";", p.Database, admin.Username)
	delete := fmt.Sprintf("DROP DATABASE \"%s\";", p.Database)

	if p.isDbExist(admin) {
		err := p.executeExec("postgres", revoke, admin)
		if err != nil {
			logrus.Errorf("failed revoking connection on database %s - %s", revoke, err)
			return err
		}

		err = kci.Retry(3, 5*time.Second, func() error {
			err := p.executeExec("postgres", delete, admin)
			if err != nil {
				// This error will result in a retry
				logrus.Debugf("failed error: %s...retry...", err)
				return err
			}

			return nil
		})
		if err != nil {
			logrus.Debugf("retry failed  %s", err)
			return err
		}
	}
	return nil
}

func (p Postgres) deleteUser(admin AdminCredentials) error {
	delete := fmt.Sprintf("DROP USER \"%s\";", p.User)

	if p.isUserExist(admin) {
		logrus.Debugf("deleting user %s", p.User)
		err := p.executeExec("postgres", delete, admin)
		if err != nil {
			pqErr := err.(*pq.Error)
			if pqErr.Code == "2BP01" {
				// 2BP01 dependent_objects_still_exist
				logrus.Infof("%s", err)
				return nil
			}
			return err
		}
	}
	return nil
}

// GetCredentials returns credentials of the postgres database
func (p Postgres) GetCredentials() Credentials {
	return Credentials{
		Name:     p.Database,
		Username: p.User,
		Password: p.Password,
	}
}

func (p Postgres) addExtensions(admin AdminCredentials) error {
	for _, ext := range p.Extensions {
		query := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS \"%s\";", ext)
		err := p.executeExec(p.Database, query, admin)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p Postgres) enableMonitoring(admin AdminCredentials) error {
	monitoringExtension := "pg_stat_statements"

	query := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS \"%s\";", monitoringExtension)
	err := p.executeExec(p.Database, query, admin)
	if err != nil {
		return err
	}

	return nil
}

func (p Postgres) checkExtensions() error {
	for _, ext := range p.Extensions {
		query := fmt.Sprintf("SELECT 1 FROM pg_extension WHERE extname = '%s';", ext)
		if !p.isRowExist(p.Database, query, p.User, p.Password) {
			return fmt.Errorf("couldn't find extension %s in database %s", ext, p.Database)
		}
	}

	return nil
}

func (p Postgres) GetDatabaseAddress() DatabaseAddress {
	return DatabaseAddress{
		Host: p.Host,
		Port: p.Port,
	}
}

// ParseAdminCredentials parse admin username and password of postgres database from secret data
// If "user" key is not defined, take "postgres" as admin user by default
func (p Postgres) ParseAdminCredentials(data map[string][]byte) (AdminCredentials, error) {
	cred := AdminCredentials{}

	_, ok := data["user"]
	if ok {
		cred.Username = string(data["user"])
	} else {
		// default admin username is "postgres"
		cred.Username = "postgres"
	}

	// if "password" key is defined in data, take value as password
	_, ok = data["password"]
	if ok {
		cred.Password = string(data["password"])
		return cred, nil
	}

	// take value of "postgresql-password" key as password if "password" key is not defined in data
	// it's compatible with secret created by stable postgres chart
	_, ok = data["postgresql-password"]
	if ok {
		cred.Password = string(data["postgresql-password"])
		return cred, nil
	}

	// take value of "postgresql-password" key as password if "postgresql-password" and "password" key is not defined in data
	// it's compatible with secret created by stable postgres chart
	_, ok = data["postgresql-postgres-password"]
	if ok {
		cred.Password = string(data["postgresql-postgres-password"])
		return cred, nil
	}

	return cred, errors.New("can not find postgres admin credentials")
}

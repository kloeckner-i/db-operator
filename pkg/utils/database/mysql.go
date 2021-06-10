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

	"github.com/kloeckner-i/db-operator/pkg/utils/kci"

	// do not delete
	_ "github.com/go-sql-driver/mysql"

	"github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/mysql"
	"github.com/sirupsen/logrus"
)

// Mysql is a database interface, abstraced object
// represents a database on mysql instance
// can be used to execute query to mysql database
type Mysql struct {
	Backend      string
	Host         string
	Port         uint16
	Database     string
	User         string
	Password     string
	SSLEnabled   bool
	SkipCAVerify bool
}

const mysqlDefaultSSLMode = "preferred"

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

// CheckStatus checks status of mysql database
// if the connection to database works
func (m Mysql) CheckStatus() error {
	db, err := m.getDbConn(m.User, m.Password)
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

func (m Mysql) getDbConn(user, password string) (*sql.DB, error) {
	var db *sql.DB
	var err error

	switch m.Backend {
	case "google":
		db, err = mysql.DialPassword(m.Host, user, password)
		if err != nil {
			logrus.Debugf("failed to validate db connection: %s", err)
			return db, err
		}
	default:
		dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%d)/?tls=%s", user, password, m.Host, m.Port, m.sslMode())
		db, err = sql.Open("mysql", dataSourceName)
		if err != nil {
			logrus.Debugf("failed to validate db connection: %s", err)
			return db, err
		}
		db.SetMaxIdleConns(0)
	}

	return db, nil
}

func (m Mysql) executeQuery(query string, admin AdminCredentials) error {
	db, err := m.getDbConn(admin.Username, admin.Password)
	if err != nil {
		logrus.Fatalf("failed to get db connection: %s", err)
	}

	defer db.Close()
	_, err = db.Query(query)
	if err != nil {
		logrus.Debugf("failed to execute query: %s", err)
		return err
	}

	return nil
}

func (m Mysql) createDatabase(admin AdminCredentials) error {
	create := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`;", m.Database)

	err := m.executeQuery(create, admin)
	if err != nil {
		return err
	}

	return nil
}

func (m Mysql) deleteDatabase(admin AdminCredentials) error {
	create := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", m.Database)

	err := kci.Retry(3, 5*time.Second, func() error {
		err := m.executeQuery(create, admin)
		if err != nil {
			logrus.Debugf("failed error: %s...retry...", err)
			return err
		}

		return nil
	})
	if err != nil {
		logrus.Debugf("retry failed  %s", err)
		return err
	}

	return nil
}

func (m Mysql) createUser(admin AdminCredentials) error {
	create := fmt.Sprintf("CREATE USER `%s` IDENTIFIED BY '%s';", m.User, m.Password)
	grant := fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%';", m.Database, m.User)
	update := fmt.Sprintf("SET PASSWORD FOR `%s` = PASSWORD('%s');", m.User, m.Password)

	if !m.isUserExist(admin) {
		err := m.executeQuery(create, admin)
		if err != nil {
			return err
		}
	} else {
		err := m.executeQuery(update, admin)
		if err != nil {
			return err
		}
	}

	err := m.executeQuery(grant, admin)
	if err != nil {
		return err
	}

	return nil
}

func (m Mysql) deleteUser(admin AdminCredentials) error {
	delete := fmt.Sprintf("DROP USER `%s`;", m.User)

	if m.isUserExist(admin) {
		err := m.executeQuery(delete, admin)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m Mysql) isRowExist(query string, admin AdminCredentials) bool {
	db, err := m.getDbConn(admin.Username, admin.Password)
	if err != nil {
		logrus.Fatalf("failed to get db connection: %s", err)
	}
	defer db.Close()

	var result string
	err = db.QueryRow(query).Scan(&result)
	if err != nil {
		logrus.Debug(err)
		return false
	}

	return true
}

func (m Mysql) isUserExist(admin AdminCredentials) bool {
	check := fmt.Sprintf("SELECT User FROM mysql.user WHERE user='%s';", m.User)

	if m.isRowExist(check, admin) {
		logrus.Debug("user exists")
		return true
	}

	logrus.Debug("user doesn't exists")
	return false
}

// GetCredentials returns credentials of the mysql database
func (m Mysql) GetCredentials() Credentials {

	return Credentials{
		Name:     m.Database,
		Username: m.User,
		Password: m.Password,
	}
}

// ParseAdminCredentials parse admin username and password of mysql database from secret data
// If "user" key is not defined, take "root" as admin user by default
func (m Mysql) ParseAdminCredentials(data map[string][]byte) (AdminCredentials, error) {
	cred := AdminCredentials{}

	_, ok := data["user"]
	if ok {
		cred.Username = string(data["user"])
	} else {
		// default admin username is "root"
		cred.Username = "root"
	}

	// if "password" key is defined in data, take value as password
	_, ok = data["password"]
	if ok {
		cred.Password = string(data["password"])
		return cred, nil
	}

	// take value of "mysql-root-password" key as password if "password" key is not defined in data
	// it's compatible with secret created by stable mysql chart
	_, ok = data["mysql-root-password"]
	if ok {
		cred.Password = string(data["mysql-root-password"])
		return cred, nil
	}

	return cred, errors.New("can not find mysql admin credentials")
}

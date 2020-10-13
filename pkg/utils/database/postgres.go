package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/kloeckner-i/db-operator/pkg/utils/kci"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"

	// Don't delete below package. Used for driver "postgres"
	_ "github.com/lib/pq"
	// Don't delete below package. Used for driver "cloudsqlpostgres"
	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/postgres"
)

// Postgres is a database interface, abstraced object
// represents a database on postgres instance
// can be used to execute query to postgres database
type Postgres struct {
	Backend      string
	Host         string
	Port         uint16
	Database     string
	User         string
	Password     string
	Extensions   []string
	SSLEnabled   bool
	SkipCAVerify bool
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

	return db, err
}

func (p Postgres) executeQuery(database, query string, admin AdminCredentials) error {
	db, err := p.getDbConn(database, admin.Username, admin.Password)
	if err != nil {
		logrus.Fatalf("failed to open db connection: %s", err)
	}

	defer db.Close()
	_, err = db.Query(query)

	return err
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

	if p.isRowExist("postgres", check, admin.Username, admin.Password) {
		return true
	}
	return false
}

func (p Postgres) isUserExist(admin AdminCredentials) bool {
	check := fmt.Sprintf("SELECT 1 FROM pg_user WHERE usename = '%s';", p.User)

	if p.isRowExist("postgres", check, admin.Username, admin.Password) {
		return true
	}
	return false
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

	err = p.checkExtensions()
	if err != nil {
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
		err := p.executeQuery("postgres", create, admin)
		if err != nil {
			logrus.Errorf("failed creating postgres database %s", err)
			return err
		}
	}

	err := p.addExtensions(admin)
	if err != nil {
		logrus.Errorf("failed creating postgres extensions %s", err)
		return err
	}

	return nil
}

func (p Postgres) createUser(admin AdminCredentials) error {
	create := fmt.Sprintf("CREATE USER \"%s\" WITH ENCRYPTED PASSWORD '%s' NOSUPERUSER;", p.User, p.Password)
	grant := fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE \"%s\" TO \"%s\";", p.Database, p.User)
	update := fmt.Sprintf("ALTER ROLE \"%s\" WITH ENCRYPTED PASSWORD '%s';", p.User, p.Password)

	if !p.isUserExist(admin) {
		err := p.executeQuery("postgres", create, admin)
		if err != nil {
			logrus.Errorf("failed creating postgres user - %s", err)
			return err
		}
	} else {
		err := p.executeQuery("postgres", update, admin)
		if err != nil {
			logrus.Errorf("failed updating postgres user %s - %s", update, err)
			return err
		}
	}

	err := p.executeQuery("postgres", grant, admin)
	if err != nil {
		logrus.Errorf("failed granting postgres user %s - %s", grant, err)
		return err
	}

	return nil
}

func (p Postgres) deleteDatabase(admin AdminCredentials) error {
	revoke := fmt.Sprintf("REVOKE CONNECT ON DATABASE \"%s\" FROM PUBLIC, \"%s\";", p.Database, admin.Username)
	delete := fmt.Sprintf("DROP DATABASE \"%s\";", p.Database)

	if p.isDbExist(admin) {
		err := p.executeQuery("postgres", revoke, admin)
		if err != nil {
			logrus.Errorf("failed revoking connection on database %s - %s", revoke, err)
			return err
		}

		err = kci.Retry(3, 5*time.Second, func() error {
			err := p.executeQuery("postgres", delete, admin)
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
		err := p.executeQuery("postgres", delete, admin)
		if err != nil {
			pqErr := err.(*pq.Error)
			if pqErr.Code == "2BP01" {
				//2BP01 dependent_objects_still_exist
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
			logrus.Errorf("failed creating extensions %s - %s", query, err)
			return err
		}
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

package database

import (
	"errors"
	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	database "github.com/kloeckner-i/db-operator/pkg/utils/database"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	"strconv"

	"github.com/sirupsen/logrus"
)

func determinDatabaseType(dbcr *kciv1alpha1.Database, dbCred database.Credentials) (database.Database, error) {
	instance, err := dbcr.GetInstanceRef()
	if err != nil {
		logrus.Errorf("could not get instance ref %s - %s", dbcr.Name, err)
		return nil, err
	}

	host := instance.Status.Info["DB_CONN"]
	port, err := strconv.Atoi(instance.Status.Info["DB_PORT"])
	if err != nil {
		logrus.Errorf("can't get port information from the instanceRef %s - %s", dbcr.Name, err)
		return nil, err
	}

	engine, err := dbcr.GetEngineType()
	if err != nil {
		logrus.Errorf("could not get instance engine type %s - %s", dbcr.Name, err)
		return nil, err
	}

	backend, err := dbcr.GetBackendType()
	if err != nil {
		logrus.Errorf("could not get backend type %s - %s", dbcr.Name, err)
		return nil, err
	}

	switch engine {
	case "postgres":
		db := database.Postgres{
			Backend:    backend,
			Host:       host,
			Port:       int32(port),
			Database:   dbCred.Name,
			User:       dbCred.Username,
			Password:   dbCred.Password,
			Extensions: dbcr.Spec.Extensions,
		}

		return db, nil

	case "mysql":
		db := database.Mysql{
			Backend:  backend,
			Host:     host,
			Port:     int32(port),
			Database: dbCred.Name,
			User:     dbCred.Username,
			Password: dbCred.Password,
		}

		return db, nil
	default:
		err := errors.New("not supported engine type")
		return nil, err
	}
}

func parseDatabaseSecretData(dbcr *kciv1alpha1.Database, data map[string][]byte) (database.Credentials, error) {
	cred := database.Credentials{}
	engine, err := dbcr.GetEngineType()
	if err != nil {
		return cred, err
	}

	switch engine {
	case "postgres":
		if name, ok := data["POSTGRES_DB"]; ok {
			cred.Name = string(name)
		} else {
			return cred, errors.New("POSTGRES_DB key does not exists in secret data")
		}

		if user, ok := data["POSTGRES_USER"]; ok {
			cred.Username = string(user)
		} else {
			return cred, errors.New("POSTGRES_USER key does not exists in secret data")
		}

		if pass, ok := data["POSTGRES_PASSWORD"]; ok {
			cred.Password = string(pass)
		} else {
			return cred, errors.New("POSTGRES_PASSWORD key does not exists in secret data")
		}

		return cred, nil
	case "mysql":
		if name, ok := data["DB"]; ok {
			cred.Name = string(name)
		} else {
			return cred, errors.New("DB key does not exists in secret data")
		}

		if user, ok := data["USER"]; ok {
			cred.Username = string(user)
		} else {
			return cred, errors.New("USER key does not exists in secret data")
		}

		if pass, ok := data["PASSWORD"]; ok {
			cred.Password = string(pass)
		} else {
			return cred, errors.New("PASSWORD key does not exists in secret data")
		}

		return cred, nil
	default:
		return cred, errors.New("not supported engine type")
	}
}

func parseDatabaseAdminSecretData(dbcr *kciv1alpha1.Database, data map[string][]byte) (database.AdminCredentials, error) {
	cred := database.AdminCredentials{}

	_, ok := data["user"]
	if !ok {
		return cred, errors.New("can not find user in admin secret data")
	}
	_, ok = data["password"]
	if !ok {
		return cred, errors.New("can not find password in admin secret data")
	}

	cred.Username = string(data["user"])
	cred.Password = string(data["password"])

	return cred, nil
}

func generateDatabaseSecretData(dbcr *kciv1alpha1.Database) (map[string][]byte, error) {
	engine, err := dbcr.GetEngineType()
	if err != nil {
		return nil, err
	}

	dbName := dbcr.Namespace + "-" + dbcr.Name
	dbUser := dbcr.Namespace + "-" + dbcr.Name
	dbPassword := kci.GeneratePass()

	switch engine {
	case "postgres":
		data := map[string][]byte{
			"POSTGRES_DB":       []byte(dbName),
			"POSTGRES_USER":     []byte(dbUser),
			"POSTGRES_PASSWORD": []byte(dbPassword)}
		return data, nil
	case "mysql":
		data := map[string][]byte{
			"DB":       []byte(stringShortner(dbName)),
			"USER":     []byte(stringShortner(dbUser)),
			"PASSWORD": []byte(dbPassword)}
		return data, nil
	default:
		return nil, errors.New("not supported engine type")
	}
}

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

package controllers

import (
	"errors"
	"strconv"

	kindav1beta1 "github.com/db-operator/db-operator/api/v1beta1"
	"github.com/db-operator/db-operator/pkg/utils/database"
	"github.com/db-operator/db-operator/pkg/utils/kci"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func determinDatabaseType(dbcr *kindav1beta1.Database, dbCred database.Credentials) (database.Database, *database.DatabaseUser, error) {
	instance, err := dbcr.GetInstanceRef()
	if err != nil {
		logrus.Errorf("could not get instance ref %s - %s", dbcr.Name, err)
		return nil, nil, err
	}

	host := instance.Status.Info["DB_CONN"]
	port, err := strconv.Atoi(instance.Status.Info["DB_PORT"])
	if err != nil {
		logrus.Errorf("can't get port information from the instanceRef %s - %s", dbcr.Name, err)
		return nil, nil, err
	}

	engine, err := dbcr.GetEngineType()
	if err != nil {
		logrus.Errorf("could not get instance engine type %s - %s", dbcr.Name, err)
		return nil, nil, err
	}

	backend, err := dbcr.GetBackendType()
	if err != nil {
		logrus.Errorf("could not get backend type %s - %s", dbcr.Name, err)
		return nil, nil, err
	}

	monitoringEnabled, err := dbcr.IsMonitoringEnabled()
	if err != nil {
		logrus.Errorf("could not check if monitoring is enabled %s - %s", dbcr.Name, err)
		return nil, nil, err
	}

	dbuser := &database.DatabaseUser{
		Username: dbCred.Username,
		Password: dbCred.Password,
	}

	switch engine {
	case "postgres":
		extList := dbcr.Spec.Postgres.Extensions
		db := database.Postgres{
			Backend:          backend,
			Host:             host,
			Port:             uint16(port),
			Database:         dbCred.Name,
			Monitoring:       monitoringEnabled,
			Extensions:       extList,
			SSLEnabled:       instance.Spec.SSLConnection.Enabled,
			SkipCAVerify:     instance.Spec.SSLConnection.SkipVerify,
			DropPublicSchema: dbcr.Spec.Postgres.DropPublicSchema,
			Schemas:          dbcr.Spec.Postgres.Schemas,
			Template:         dbcr.Spec.Postgres.Template,
		}
		return db, dbuser, nil

	case "mysql":
		db := database.Mysql{
			Backend:      backend,
			Host:         host,
			Port:         uint16(port),
			Database:     dbCred.Name,
			SSLEnabled:   instance.Spec.SSLConnection.Enabled,
			SkipCAVerify: instance.Spec.SSLConnection.SkipVerify,
		}

		return db, dbuser, nil
	default:
		err := errors.New("not supported engine type")
		return nil, nil, err
	}
}

func parseDatabaseSecretData(dbcr *kindav1beta1.Database, data map[string][]byte) (database.Credentials, error) {
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
			return cred, errors.New("POSTGRES_DB key does not exist in secret data")
		}

		if user, ok := data["POSTGRES_USER"]; ok {
			cred.Username = string(user)
		} else {
			return cred, errors.New("POSTGRES_USER key does not exist in secret data")
		}

		if pass, ok := data["POSTGRES_PASSWORD"]; ok {
			cred.Password = string(pass)
		} else {
			return cred, errors.New("POSTGRES_PASSWORD key does not exist in secret data")
		}

		return cred, nil
	case "mysql":
		if name, ok := data["DB"]; ok {
			cred.Name = string(name)
		} else {
			return cred, errors.New("DB key does not exist in secret data")
		}

		if user, ok := data["USER"]; ok {
			cred.Username = string(user)
		} else {
			return cred, errors.New("USER key does not exist in secret data")
		}

		if pass, ok := data["PASSWORD"]; ok {
			cred.Password = string(pass)
		} else {
			return cred, errors.New("PASSWORD key does not exist in secret data")
		}

		return cred, nil
	default:
		return cred, errors.New("not supported engine type")
	}
}

	func generateDatabaseSecretData(dbcr metav1.ObjectMeta, engine string) (map[string][]byte, error) {
	const (
		// https://dev.mysql.com/doc/refman/5.7/en/identifier-length.html
		mysqlDBNameLengthLimit = 63
		// https://dev.mysql.com/doc/refman/5.7/en/replication-features-user-names.html
		mysqlUserLengthLimit = 32
	)

	dbName := dbcr.Namespace + "-" + dbcr.Name
	dbUser := dbcr.Namespace + "-" + dbcr.Name
	dbPassword := kci.GeneratePass()

	switch engine {
	case "postgres":
		data := map[string][]byte{
			"POSTGRES_DB":       []byte(dbName),
			"POSTGRES_USER":     []byte(dbUser),
			"POSTGRES_PASSWORD": []byte(dbPassword),
		}
		return data, nil
	case "mysql":
		data := map[string][]byte{
			"DB":       []byte(kci.StringSanitize(dbName, mysqlDBNameLengthLimit)),
			"USER":     []byte(kci.StringSanitize(dbUser, mysqlUserLengthLimit)),
			"PASSWORD": []byte(dbPassword),
		}
		return data, nil
	default:
		return nil, errors.New("not supported engine type")
	}
}

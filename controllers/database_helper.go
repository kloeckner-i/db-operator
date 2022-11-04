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
	"bytes"
	"errors"
	"strconv"
	"text/template"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/api/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/utils/database"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

// ConnectionStringFields defines default fields that can be used to generate a connection string
// / Deprecated
type ConnectionStringFields struct {
	Protocol     string
	DatabaseHost string
	DatabasePort int32
	UserName     string
	Password     string
	DatabaseName string
}

// SecretsTemplatesFields defines default fields that can be used to generate secrets with db creds
type SecretsTemplatesFields struct {
	Protocol     string
	DatabaseHost string
	DatabasePort int32
	UserName     string
	Password     string
	DatabaseName string
}

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

	monitoringEnabled, err := dbcr.IsMonitoringEnabled()
	if err != nil {
		logrus.Errorf("could not check if monitoring is enabled %s - %s", dbcr.Name, err)
		return nil, err
	}

	switch engine {
	case "postgres":
		extList := dbcr.Spec.Extensions
		db := database.Postgres{
			Backend:          backend,
			Host:             host,
			Port:             uint16(port),
			Database:         dbCred.Name,
			User:             dbCred.Username,
			Password:         dbCred.Password,
			Monitoring:       monitoringEnabled,
			Extensions:       extList,
			SSLEnabled:       instance.Spec.SSLConnection.Enabled,
			SkipCAVerify:     instance.Spec.SSLConnection.SkipVerify,
			DropPublicSchema: dbcr.Spec.Postgres.DropPublicSchema,
			Schemas:          dbcr.Spec.Postgres.Schemas,
		}

		return db, nil

	case "mysql":
		db := database.Mysql{
			Backend:      backend,
			Host:         host,
			Port:         uint16(port),
			Database:     dbCred.Name,
			User:         dbCred.Username,
			Password:     dbCred.Password,
			SSLEnabled:   instance.Spec.SSLConnection.Enabled,
			SkipCAVerify: instance.Spec.SSLConnection.SkipVerify,
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

	// Connection string can be empty
	if connectionString, ok := data["CONNECTION_STRING"]; ok {
		cred.ConnectionString = string(connectionString)
	} else {
		logrus.Info("CONNECTION_STRING key does not exist in secret data")
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

func generateDatabaseSecretData(dbcr *kciv1alpha1.Database) (map[string][]byte, error) {
	const (
		// https://dev.mysql.com/doc/refman/5.7/en/identifier-length.html
		mysqlDBNameLengthLimit = 63
		// https://dev.mysql.com/doc/refman/5.7/en/replication-features-user-names.html
		mysqlUserLengthLimit = 32
	)

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

func generateConnectionString(dbcr *kciv1alpha1.Database, databaseCred database.Credentials) (connString string, err error) {
	// The string that's going to be generated if the default template is used:
	// "postgresql://user:password@host:port/database"
	const defaultTemplate = "{{ .Protocol }}://{{ .UserName }}:{{ .Password }}@{{ .DatabaseHost }}:{{ .DatabasePort }}/{{ .DatabaseName }}"

	dbData := ConnectionStringFields{
		DatabaseHost: dbcr.Status.ProxyStatus.ServiceName,
		DatabasePort: dbcr.Status.ProxyStatus.SQLPort,
		UserName:     databaseCred.Username,
		Password:     databaseCred.Password,
		DatabaseName: databaseCred.Name,
	}

	// If proxy is not used, set a real database address
	if !dbcr.Status.ProxyStatus.Status {
		db, err := determinDatabaseType(dbcr, databaseCred)
		if err != nil {
			return "", err
		}
		dbAddress := db.GetDatabaseAddress()
		dbData.DatabaseHost = dbAddress.Host
		dbData.DatabasePort = int32(dbAddress.Port)
	}

	// If engine is 'postgres', the protocol should be postgresql
	if dbcr.Status.InstanceRef.Spec.Engine == "postgres" {
		dbData.Protocol = "postgresql"
	} else {
		dbData.Protocol = dbcr.Status.InstanceRef.Spec.Engine
	}

	// If dbcr.Spec.ConnectionString is not specified, use the defalt template
	var tmpl string
	if dbcr.Spec.ConnectionStringTemplate != "" {
		tmpl = dbcr.Spec.ConnectionStringTemplate
	} else {
		tmpl = defaultTemplate
	}

	t, err := template.New("connection_string").Parse(tmpl)
	if err != nil {
		logrus.Error(err)
		return
	}

	var connStringBytes bytes.Buffer
	err = t.Execute(&connStringBytes, dbData)
	if err != nil {
		logrus.Error(err)
		return
	}

	connString = connStringBytes.String()
	return
}

func generateTemplatedSecrets(dbcr *kciv1alpha1.Database, databaseCred database.Credentials) (secrets map[string]string, err error) {
	secrets = map[string]string{}
	// The string that's going to be generated if the default template is used:
	// "postgresql://user:password@host:port/database"
	dbData := ConnectionStringFields{
		DatabaseHost: dbcr.Status.ProxyStatus.ServiceName,
		DatabasePort: dbcr.Status.ProxyStatus.SQLPort,
		UserName:     databaseCred.Username,
		Password:     databaseCred.Password,
		DatabaseName: databaseCred.Name,
	}

	// If proxy is not used, set a real database address
	if !dbcr.Status.ProxyStatus.Status {
		db, err := determinDatabaseType(dbcr, databaseCred)
		if err != nil {
			return nil, err
		}
		dbAddress := db.GetDatabaseAddress()
		dbData.DatabaseHost = dbAddress.Host
		dbData.DatabasePort = int32(dbAddress.Port)
	}
	// If engine is 'postgres', the protocol should be postgresql
	if dbcr.Status.InstanceRef.Spec.Engine == "postgres" {
		dbData.Protocol = "postgresql"
	} else {
		dbData.Protocol = dbcr.Status.InstanceRef.Spec.Engine
	}

	// If dbcr.Spec.ConnectionString is not specified, use the defalt template
	if len(dbcr.Spec.SecretsTemplates) > 0 {
		for key, value := range dbcr.Spec.SecretsTemplates {
			var tmpl string = value
			t, err := template.New("secret").Parse(tmpl)
			if err != nil {
				logrus.Error(err)
				return nil, err
			}

			var secretBytes bytes.Buffer
			err = t.Execute(&secretBytes, dbData)
			if err != nil {
				logrus.Error(err)
				return nil, err
			}

			connString := secretBytes.String()

			secrets[key] = connString
		}
	} else {
		const tmpl = "{{ .Protocol }}://{{ .UserName }}:{{ .Password }}@{{ .DatabaseHost }}:{{ .DatabasePort }}/{{ .DatabaseName }}"
		t, err := template.New("secret").Parse(tmpl)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}

		var secretBytes bytes.Buffer
		err = t.Execute(&secretBytes, dbData)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}

		connString := secretBytes.String()

		secrets["CONNECTION_STRING"] = connString
	}
	return
}

func addConnectionStringToSecret(dbcr *kciv1alpha1.Database, secretData map[string][]byte, connectionString string) *v1.Secret {
	secretData["CONNECTION_STRING"] = []byte(connectionString)
	return kci.SecretBuilder(dbcr.Spec.SecretName, dbcr.GetNamespace(), secretData)
}

func addTemplatedSecretToSecret(dbcr *kciv1alpha1.Database, secretData map[string][]byte, secretName string, secretValue string) *v1.Secret {
	secretData[secretName] = []byte(secretValue)
	return kci.SecretBuilder(dbcr.Spec.SecretName, dbcr.GetNamespace(), secretData)
}

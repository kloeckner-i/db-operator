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

	kciv1beta1 "github.com/kloeckner-i/db-operator/api/v1beta1"
	"github.com/kloeckner-i/db-operator/pkg/utils/database"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/strings/slices"
)

// SecretsTemplatesFields defines default fields that can be used to generate secrets with db creds
type SecretsTemplatesFields struct {
	Protocol     string
	DatabaseHost string
	DatabasePort int32
	UserName     string
	Password     string
	DatabaseName string
}

const (
	fieldPostgresDB        = "POSTGRES_DB"
	fieldPostgresUser      = "POSTGRES_USER"
	fieldPostgressPassword = "POSTGRES_PASSWORD"
	fieldMysqlDB           = "DB"
	fieldMysqlUser         = "USER"
	fieldMysqlPassword     = "PASSWORD"
)

func getBlockedTempatedKeys() []string {
	return []string{fieldMysqlDB, fieldMysqlPassword, fieldMysqlUser, fieldPostgresDB, fieldPostgresUser, fieldPostgressPassword}
}

func determinDatabaseType(dbcr *kciv1beta1.Database, dbCred database.Credentials) (database.Database, error) {
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
		extList := dbcr.Spec.Postgres.Extensions
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

func parseTemplatedSecretsData(dbcr *kciv1beta1.Database, data map[string][]byte) (database.Credentials, error) {
	cred, err := parseDatabaseSecretData(dbcr, data)
	if err != nil {
		return cred, err
	}
	cred.TemplatedSecrets = map[string]string{}
	for key := range dbcr.Spec.SecretsTemplates {
		// Here we can see if there are obsolete entries in the secret data
		if secret, ok := data[key]; ok {
			delete(data, key)
			cred.TemplatedSecrets[key] = string(secret)
		} else {
			logrus.Infof("DB: namespace=%s, name=%s %s key does not exist in secret data",
				dbcr.Namespace,
				dbcr.Name,
				key,
			)
		}
	}

	return cred, nil
}

func parseDatabaseSecretData(dbcr *kciv1beta1.Database, data map[string][]byte) (database.Credentials, error) {
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

func generateDatabaseSecretData(dbcr *kciv1beta1.Database) (map[string][]byte, error) {
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

func generateTemplatedSecrets(dbcr *kciv1beta1.Database, databaseCred database.Credentials) (secrets map[string]string, err error) {
	secrets = map[string]string{}
	templates := map[string]string{}
	if len(dbcr.Spec.SecretsTemplates) > 0 {
		templates = dbcr.Spec.SecretsTemplates
	} else {
		const tmpl = "{{ .Protocol }}://{{ .UserName }}:{{ .Password }}@{{ .DatabaseHost }}:{{ .DatabasePort }}/{{ .DatabaseName }}"
		templates["CONNECTION_STRING"] = tmpl
	}
	// The string that's going to be generated if the default template is used:
	// "postgresql://user:password@host:port/database"
	dbData := SecretsTemplatesFields{
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

	logrus.Infof("DB: namespace=%s, name=%s creating secrets from templates", dbcr.Namespace, dbcr.Name)
	for key, value := range templates {
		var tmpl string = value
		t, err := template.New("secret").Parse(tmpl)
		if err != nil {
			return nil, err
		}

		var secretBytes bytes.Buffer
		err = t.Execute(&secretBytes, dbData)
		if err != nil {
			return nil, err
		}
		connString := secretBytes.String()
		secrets[key] = connString
	}
	return secrets, nil
}

func fillTemplatedSecretData(dbcr *kciv1beta1.Database, secretData map[string][]byte, newSecretFields map[string]string, ownership []metav1.OwnerReference) (newSecret *v1.Secret) {
	blockedTempatedKeys := getBlockedTempatedKeys()
	for key, value := range newSecretFields {
		if slices.Contains(blockedTempatedKeys, key) {
			logrus.Warnf("DB: namespace=%s, name=%s %s can't be used for templating, because it's used for default secret created by operator",
				dbcr.Namespace,
				dbcr.Name,
				key,
			)
		} else {
			newSecret = addTemplatedSecretToSecret(dbcr, secretData, key, value, ownership)
		}
	}
	return
}

func addTemplatedSecretToSecret(dbcr *kciv1beta1.Database, secretData map[string][]byte, secretName string, secretValue string, ownership []metav1.OwnerReference) *v1.Secret {
	secretData[secretName] = []byte(secretValue)
	return kci.SecretBuilder(dbcr.Spec.SecretName, dbcr.GetNamespace(), secretData, ownership)
}

func removeObsoleteSecret(dbcr *kciv1beta1.Database, secretData map[string][]byte, newSecretFields map[string]string, ownership []metav1.OwnerReference) *v1.Secret {
	blockedTempatedKeys := getBlockedTempatedKeys()

	for key := range secretData {
		if _, ok := newSecretFields[key]; !ok {
			// Check if is a untemplatead secret, so it's not removed accidentally
			if !slices.Contains(blockedTempatedKeys, key) {
				logrus.Infof("DB: namespace=%s, name=%s removing an obsolete field: %s", dbcr.Namespace, dbcr.Name, key)
				delete(secretData, key)
			}
		}
	}

	return kci.SecretBuilder(dbcr.Spec.SecretName, dbcr.GetNamespace(), secretData, ownership)
}

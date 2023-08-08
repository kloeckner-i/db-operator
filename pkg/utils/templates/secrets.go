/*
 * Copyright 2021 kloeckner.i GmbH
 * Copyright 2023 Nikolai Rodionov (allanger)
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

package templates

import (
	"bytes"
	"text/template"

	kindav1beta1 "github.com/db-operator/db-operator/api/v1beta1"
	"github.com/db-operator/db-operator/pkg/utils/database"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/strings/slices"
)

const (
	FieldPostgresDB        = "POSTGRES_DB"
	FieldPostgresUser      = "POSTGRES_USER"
	FieldPostgressPassword = "POSTGRES_PASSWORD"
	FieldMysqlDB           = "DB"
	FieldMysqlUser         = "USER"
	FieldMysqlPassword     = "PASSWORD"
)

const (
	defaultTemplate = "{{ .Protocol }}://{{ .UserName }}:{{ .Password }}@{{ .DatabaseHost }}:{{ .DatabasePort }}/{{ .DatabaseName }}"
	defaultKey      = "CONNECTION_STRING"
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

// This function is blocking the secretTemplates feature from
//
//	templating fields that are created by the operator by default
func getBlockedTempatedKeys() []string {
	return []string{FieldMysqlDB, FieldMysqlPassword, FieldMysqlUser, FieldPostgresDB, FieldPostgresUser, FieldPostgressPassword}
}

func ParseTemplatedSecretsData(dbcr *kindav1beta1.Database, cred database.Credentials, data map[string][]byte) (database.Credentials, error) {
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

func GenerateTemplatedSecrets(dbcr *kindav1beta1.Database, databaseCred database.Credentials, dbAddress database.DatabaseAddress) (secrets map[string][]byte, err error) {
	secrets = map[string][]byte{}
	templates := map[string]string{}
	if len(dbcr.Spec.SecretsTemplates) > 0 {
		templates = dbcr.Spec.SecretsTemplates
	} else {
		templates[defaultKey] = defaultTemplate
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
		dbAddress := dbAddress
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
		if slices.Contains(getBlockedTempatedKeys(), key) {
			logrus.Warnf("DB: namespace=%s, name=%s %s can't be used for templating, because it's used for default secret created by operator",
				dbcr.Namespace,
				dbcr.Name,
				key,
			)
		} else {
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
			templatedSecret := secretBytes.String()
			secrets[key] = []byte(templatedSecret)
		}
	}
	return secrets, nil
}

func AppendTemplatedSecretData(dbcr *kindav1beta1.Database, secretData map[string][]byte, newSecretFields map[string][]byte, ownership []metav1.OwnerReference) map[string][]byte {
	blockedTempatedKeys := getBlockedTempatedKeys()
	for key, value := range newSecretFields {
		if slices.Contains(blockedTempatedKeys, key) {
			logrus.Warnf("DB: namespace=%s, name=%s %s can't be used for templating, because it's used for default secret created by operator",
				dbcr.Namespace,
				dbcr.Name,
				key,
			)
		} else {
			secretData[key] = value
		}
	}
	return secretData
}

func RemoveObsoleteSecret(dbcr *kindav1beta1.Database, secretData map[string][]byte, newSecretFields map[string][]byte, ownership []metav1.OwnerReference) map[string][]byte {
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
	return secretData
}

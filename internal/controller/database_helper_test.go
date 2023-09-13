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

package controllers

import (
	"fmt"
	"testing"

	"github.com/db-operator/db-operator/pkg/utils/database"
	"github.com/db-operator/db-operator/pkg/utils/templates"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	testDbcred = database.Credentials{Name: "testdb", Username: "testuser", Password: "password"}
	ownership  = []metav1.OwnerReference{}
)

func TestUnitDeterminPostgresType(t *testing.T) {
	postgresDbCr := newPostgresTestDbCr(newPostgresTestDbInstanceCr())

	db, _, _ := determinDatabaseType(postgresDbCr, testDbcred)
	_, ok := db.(database.Postgres)
	assert.Equal(t, ok, true, "expected true")
}

func TestUnitDeterminMysqlType(t *testing.T) {
	mysqlDbCr := newMysqlTestDbCr()

	db, _, _ := determinDatabaseType(mysqlDbCr, testDbcred)
	_, ok := db.(database.Mysql)
	assert.Equal(t, ok, true, "expected true")
}

func TestUnitParsePostgresSecretData(t *testing.T) {
	postgresDbCr := newPostgresTestDbCr(newPostgresTestDbInstanceCr())

	invalidData := make(map[string][]byte)
	invalidData["DB"] = []byte("testdb")

	_, err := parseDatabaseSecretData(postgresDbCr, invalidData)
	assert.Errorf(t, err, "should get error %v", err)

	validData := make(map[string][]byte)
	validData["POSTGRES_DB"] = []byte("testdb")
	validData["POSTGRES_USER"] = []byte("testuser")
	validData["POSTGRES_PASSWORD"] = []byte("testpassword")

	cred, err := parseDatabaseSecretData(postgresDbCr, validData)
	assert.NoErrorf(t, err, "expected no error %v", err)
	assert.Equal(t, string(validData["POSTGRES_DB"]), cred.Name, "expect same values")
	assert.Equal(t, string(validData["POSTGRES_USER"]), cred.Username, "expect same values")
	assert.Equal(t, string(validData["POSTGRES_PASSWORD"]), cred.Password, "expect same values")
}

func TestUnitParseMysqlSecretData(t *testing.T) {
	mysqlDbCr := newMysqlTestDbCr()

	invalidData := make(map[string][]byte)
	invalidData["DB"] = []byte("testdb")

	_, err := parseDatabaseSecretData(mysqlDbCr, invalidData)
	assert.Errorf(t, err, "should get error %v", err)

	validData := make(map[string][]byte)
	validData["DB"] = []byte("testdb")
	validData["USER"] = []byte("testuser")
	validData["PASSWORD"] = []byte("testpassword")

	cred, err := parseDatabaseSecretData(mysqlDbCr, validData)
	assert.NoErrorf(t, err, "expected no error %v", err)
	assert.Equal(t, string(validData["DB"]), cred.Name, "expect same values")
	assert.Equal(t, string(validData["USER"]), cred.Username, "expect same values")
	assert.Equal(t, string(validData["PASSWORD"]), cred.Password, "expect same values")
}

func TestUnitMonitoringNotEnabled(t *testing.T) {
	instance := newPostgresTestDbInstanceCr()
	instance.Spec.Monitoring.Enabled = false
	postgresDbCr := newPostgresTestDbCr(instance)
	db, _, _ := determinDatabaseType(postgresDbCr, testDbcred)
	postgresInterface, _ := db.(database.Postgres)

	found := false
	for _, ext := range postgresInterface.Extensions {
		if ext == "pg_stat_statements" {
			found = true
			break
		}
	}
	assert.Equal(t, found, false, "expected pg_stat_statement is not included in extension list")
}

func TestUnitMonitoringEnabled(t *testing.T) {
	instance := newPostgresTestDbInstanceCr()
	instance.Spec.Monitoring.Enabled = false
	postgresDbCr := newPostgresTestDbCr(instance)
	postgresDbCr.Status.InstanceRef.Spec.Monitoring.Enabled = true

	db, _, _ := determinDatabaseType(postgresDbCr, testDbcred)
	postgresInterface, _ := db.(database.Postgres)

	assert.Equal(t, postgresInterface.Monitoring, true, "expected monitoring is true in postgres interface")
}

func TestUnitPsqlTemplatedSecretGeneratationWithProxy(t *testing.T) {
	instance := newPostgresTestDbInstanceCr()
	postgresDbCr := newPostgresTestDbCr(instance)
	postgresDbCr.Status.ProxyStatus.Status = true
	postgresDbCr.Spec.SecretsTemplates = map[string]string {
		"PROXIED_HOST": "{{ .DatabaseHost }}",
	}

	c := templates.SecretsTemplatesFields{
		DatabaseHost: "proxied_host",
		DatabasePort: 5432,
		UserName:     testDbcred.Username,
		Password:     testDbcred.Password,
		DatabaseName: testDbcred.Name,
	}

	postgresDbCr.Status.ProxyStatus.SQLPort = c.DatabasePort
	postgresDbCr.Status.ProxyStatus.ServiceName = c.DatabaseHost


	expectedData := map[string][]byte{
		"PROXIED_HOST": []byte(c.DatabaseHost),
	}

	db, _, _ := determinDatabaseType(postgresDbCr, testDbcred)
	connString, err := templates.GenerateTemplatedSecrets(postgresDbCr, testDbcred, db.GetDatabaseAddress())
	if err != nil {
		t.Logf("Unexpected error: %s", err)
		t.Fail()
	}
	assert.Equal(t, expectedData, connString, "generated connections string is wrong")
}

func TestUnitPsqlCustomSecretGeneratation(t *testing.T) {
	instance := newPostgresTestDbInstanceCr()
	postgresDbCr := newPostgresTestDbCr(instance)

	prefix := "custom->"
	postfix := "<-for_storing_data_you_know"
	postgresDbCr.Spec.SecretsTemplates = map[string]string{
		"CHECK_1": fmt.Sprintf("%s{{ .Protocol }}://{{ .UserName }}:{{ .Password }}@{{ .DatabaseHost }}:{{ .DatabasePort }}/{{ .DatabaseName }}%s", prefix, postfix),
		"CHECK_2": "{{ .Protocol }}://{{ .UserName }}:{{ .Password }}@{{ .DatabaseHost }}:{{ .DatabasePort }}/{{ .DatabaseName }}",
	}

	c := templates.SecretsTemplatesFields{
		DatabaseHost: "postgres",
		DatabasePort: 5432,
		UserName:     testDbcred.Username,
		Password:     testDbcred.Password,
		DatabaseName: testDbcred.Name,
	}
	protocol := "postgresql"
	expectedData := map[string][]byte{
		"CHECK_1": []byte(fmt.Sprintf("%s%s://%s:%s@%s:%d/%s%s", prefix, protocol, c.UserName, c.Password, c.DatabaseHost, c.DatabasePort, c.DatabaseName, postfix)),
		"CHECK_2": []byte(fmt.Sprintf("%s://%s:%s@%s:%d/%s", protocol, c.UserName, c.Password, c.DatabaseHost, c.DatabasePort, c.DatabaseName)),
	}

	db, _, _ := determinDatabaseType(postgresDbCr, testDbcred)
	templatedSecrets, err := templates.GenerateTemplatedSecrets(postgresDbCr, testDbcred, db.GetDatabaseAddress())
	if err != nil {
		t.Logf("unexpected error: %s", err)
		t.Fail()
	}
	assert.Equal(t, templatedSecrets, expectedData, "generated connections string is wrong")
}

func TestUnitWrongTemplatedSecretGeneratation(t *testing.T) {
	instance := newPostgresTestDbInstanceCr()
	postgresDbCr := newPostgresTestDbCr(instance)

	postgresDbCr.Spec.SecretsTemplates = map[string]string{
		"TMPL": "{{ .Protocol }}://{{ .User }}:{{ .Password }}@{{ .DatabaseHost }}:{{ .DatabasePort }}/{{ .DatabaseName }}",
	}

	db, _, _ := determinDatabaseType(postgresDbCr, testDbcred)
	_, err := templates.GenerateTemplatedSecrets(postgresDbCr, testDbcred, db.GetDatabaseAddress())
	errSubstr := "can't evaluate field User in type templates.SecretsTemplatesFields"

	assert.Contains(t, err.Error(), errSubstr, "the error doesn't contain expected substring")
}

func TestUnitBlockedTempatedKeysGeneratation(t *testing.T) {
	instance := newPostgresTestDbInstanceCr()
	postgresDbCr := newPostgresTestDbCr(instance)

	postgresDbCr.Spec.SecretsTemplates = map[string]string{}
	untemplatedFields := []string{templates.FieldMysqlDB, templates.FieldMysqlPassword, templates.FieldMysqlUser, templates.FieldPostgresDB, templates.FieldPostgresUser, templates.FieldPostgressPassword}
	for _, key := range untemplatedFields {
		postgresDbCr.Spec.SecretsTemplates[key] = "DUMMY"
	}
	postgresDbCr.Spec.SecretsTemplates["TMPL"] = "DUMMY"
	expectedData := map[string][]byte{
		"TMPL": []byte("DUMMY"),
	}
	db, _, _ := determinDatabaseType(postgresDbCr, testDbcred)
	sercretData, err := templates.GenerateTemplatedSecrets(postgresDbCr, testDbcred, db.GetDatabaseAddress())
	if err != nil {
		t.Logf("unexpected error: %s", err)
		t.Fail()
	}

	dummySecret := v1.Secret{
		Data: map[string][]byte{},
	}

	newSecret := templates.AppendTemplatedSecretData(postgresDbCr, dummySecret.Data, sercretData, ownership)
	assert.Equal(t, newSecret, expectedData, "generated connections string is wrong")
}

func TestUnitObsoleteFieldsRemoving(t *testing.T) {
	instance := newPostgresTestDbInstanceCr()
	postgresDbCr := newPostgresTestDbCr(instance)

	postgresDbCr.Spec.SecretsTemplates = map[string]string{}
	untemplatedFields := []string{templates.FieldMysqlDB, templates.FieldMysqlPassword, templates.FieldMysqlUser, templates.FieldPostgresDB, templates.FieldPostgresUser, templates.FieldPostgressPassword}
	for _, key := range untemplatedFields {
		postgresDbCr.Spec.SecretsTemplates[key] = "DUMMY"
	}
	postgresDbCr.Spec.SecretsTemplates["TMPL"] = "DUMMY"
	expectedData := map[string][]byte{
		"TMPL": []byte("DUMMY"),
	}

	db, _, _ := determinDatabaseType(postgresDbCr, testDbcred)
	sercretData, err := templates.GenerateTemplatedSecrets(postgresDbCr, testDbcred, db.GetDatabaseAddress())
	if err != nil {
		t.Logf("unexpected error: %s", err)
		t.Fail()
	}

	dummySecret := v1.Secret{
		Data: map[string][]byte{
			"TO_REMOVE": []byte("this is supposed to be removed"),
		},
	}

	newSecret := templates.AppendTemplatedSecretData(postgresDbCr, dummySecret.Data, sercretData, ownership)
	newSecret = templates.RemoveObsoleteSecret(postgresDbCr, dummySecret.Data, sercretData, ownership)

	assert.Equal(t, newSecret, expectedData, "generated connections string is wrong")
}

func TestUnitGetGenericSSLModePostgres(t *testing.T) {
	posgresDbCR := newPostgresTestDbCr(newPostgresTestDbInstanceCr())
	postgresInstance, err := posgresDbCR.GetInstanceRef()
	if err != nil {
		t.Error(err)
	}

	postgresInstance.Spec.SSLConnection.Enabled = false
	postgresInstance.Spec.SSLConnection.SkipVerify = false
	mode, err := getGenericSSLMode(posgresDbCR)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, SSL_DISABLED, mode)

	postgresInstance.Spec.SSLConnection.SkipVerify = true
	mode, err = getGenericSSLMode(posgresDbCR)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, SSL_DISABLED, mode)

	postgresInstance.Spec.SSLConnection.Enabled = true
	postgresInstance.Spec.SSLConnection.SkipVerify = true
	mode, err = getGenericSSLMode(posgresDbCR)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, SSL_REQUIRED, mode)

	postgresInstance.Spec.SSLConnection.SkipVerify = false
	mode, err = getGenericSSLMode(posgresDbCR)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, SSL_VERIFY_CA, mode)
}

func TestUnitGetGenericSSLModeMysql(t *testing.T) {
	mysqlDbCR := newMysqlTestDbCr()
	mysqlInstance, err := mysqlDbCR.GetInstanceRef()
	if err != nil {
		t.Error(err)
	}

	mysqlInstance.Spec.SSLConnection.Enabled = false
	mysqlInstance.Spec.SSLConnection.SkipVerify = false
	mode, err := getGenericSSLMode(mysqlDbCR)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, SSL_DISABLED, mode)

	mysqlInstance.Spec.SSLConnection.SkipVerify = true
	mode, err = getGenericSSLMode(mysqlDbCR)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, SSL_DISABLED, mode)

	mysqlInstance.Spec.SSLConnection.Enabled = true
	mysqlInstance.Spec.SSLConnection.SkipVerify = true
	mode, err = getGenericSSLMode(mysqlDbCR)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, SSL_REQUIRED, mode)

	mysqlInstance.Spec.SSLConnection.SkipVerify = false
	mode, err = getGenericSSLMode(mysqlDbCR)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, SSL_VERIFY_CA, mode)
}
func TestUnitGetSSLModePostgres(t *testing.T) {
	posgresDbCR := newPostgresTestDbCr(newPostgresTestDbInstanceCr())
	postgresInstance, err := posgresDbCR.GetInstanceRef()
	if err != nil {
		t.Error(err)
	}

	postgresInstance.Spec.SSLConnection.Enabled = false
	postgresInstance.Spec.SSLConnection.SkipVerify = false
	mode, err := getSSLMode(posgresDbCR)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "disable", mode)

	postgresInstance.Spec.SSLConnection.SkipVerify = true
	mode, err = getSSLMode(posgresDbCR)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "disable", mode)

	postgresInstance.Spec.SSLConnection.Enabled = true
	postgresInstance.Spec.SSLConnection.SkipVerify = true
	mode, err = getSSLMode(posgresDbCR)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "require", mode)

	postgresInstance.Spec.SSLConnection.SkipVerify = false
	mode, err = getSSLMode(posgresDbCR)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "verify-ca", mode)
}

func TestUnitGetSSLModeMysql(t *testing.T) {
	mysqlDbCR := newMysqlTestDbCr()
	mysqlInstance, err := mysqlDbCR.GetInstanceRef()
	if err != nil {
		t.Error(err)
	}

	mysqlInstance.Spec.SSLConnection.Enabled = false
	mysqlInstance.Spec.SSLConnection.SkipVerify = false
	mode, err := getSSLMode(mysqlDbCR)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "disabled", mode)

	mysqlInstance.Spec.SSLConnection.SkipVerify = true
	mode, err = getSSLMode(mysqlDbCR)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "disabled", mode)

	mysqlInstance.Spec.SSLConnection.Enabled = true
	mysqlInstance.Spec.SSLConnection.SkipVerify = true
	mode, err = getSSLMode(mysqlDbCR)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "required", mode)

	mysqlInstance.Spec.SSLConnection.SkipVerify = false
	mode, err = getSSLMode(mysqlDbCR)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "verify_ca", mode)
}

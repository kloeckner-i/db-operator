/*
 * Copyright 2021 kloeckner.i GmbH
 * Copyright 2023 DB-Operator Authors
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

package database_test

import (
	"context"
	"fmt"
	"testing"

	dbhelper "github.com/db-operator/db-operator/v2/internal/helpers/database"
	"github.com/db-operator/db-operator/v2/internal/utils/testutils"
	"github.com/db-operator/db-operator/v2/pkg/consts"
	"github.com/db-operator/db-operator/v2/pkg/utils/database"
	"github.com/db-operator/db-operator/v2/pkg/utils/templates"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

var (
	testDbcred = database.Credentials{Name: "testdb", Username: "testuser", Password: "password"}
	ctx        = context.Background()
)

func TestUnitDeterminPostgresType(t *testing.T) {
	instance := testutils.NewPostgresTestDbInstanceCr()
	postgresDbCr := testutils.NewPostgresTestDbCr(instance)
	db, _, _ := dbhelper.FetchDatabaseData(ctx, postgresDbCr, testDbcred, &instance)
	_, ok := db.(database.Postgres)
	assert.Equal(t, ok, true, "expected true")
}

func TestUnitDeterminMysqlType(t *testing.T) {
	mysqlDbCr := testutils.NewMysqlTestDbCr()
	instance := testutils.NewPostgresTestDbInstanceCr()
	db, _, _ := dbhelper.FetchDatabaseData(ctx, mysqlDbCr, testDbcred, &instance)
	_, ok := db.(database.Mysql)
	assert.Equal(t, ok, true, "expected true")
}

func TestUnitParsePostgresSecretData(t *testing.T) {
	instance := testutils.NewPostgresTestDbInstanceCr()
	postgresDbCr := testutils.NewPostgresTestDbCr(instance)

	invalidData := make(map[string][]byte)
	invalidData["DB"] = []byte("testdb")

	_, err := dbhelper.ParseDatabaseSecretData(postgresDbCr, invalidData)
	assert.Errorf(t, err, "should get error %v", err)

	validData := make(map[string][]byte)
	validData["POSTGRES_DB"] = []byte("testdb")
	validData["POSTGRES_USER"] = []byte("testuser")
	validData["POSTGRES_PASSWORD"] = []byte("testpassword")

	cred, err := dbhelper.ParseDatabaseSecretData(postgresDbCr, validData)
	assert.NoErrorf(t, err, "expected no error %v", err)
	assert.Equal(t, string(validData["POSTGRES_DB"]), cred.Name, "expect same values")
	assert.Equal(t, string(validData["POSTGRES_USER"]), cred.Username, "expect same values")
	assert.Equal(t, string(validData["POSTGRES_PASSWORD"]), cred.Password, "expect same values")
}

func TestUnitParseMysqlSecretData(t *testing.T) {
	mysqlDbCr := testutils.NewMysqlTestDbCr()

	invalidData := make(map[string][]byte)
	invalidData["DB"] = []byte("testdb")

	_, err := dbhelper.ParseDatabaseSecretData(mysqlDbCr, invalidData)
	assert.Errorf(t, err, "should get error %v", err)

	validData := make(map[string][]byte)
	validData["DB"] = []byte("testdb")
	validData["USER"] = []byte("testuser")
	validData["PASSWORD"] = []byte("testpassword")

	cred, err := dbhelper.ParseDatabaseSecretData(mysqlDbCr, validData)
	assert.NoErrorf(t, err, "expected no error %v", err)
	assert.Equal(t, string(validData["DB"]), cred.Name, "expect same values")
	assert.Equal(t, string(validData["USER"]), cred.Username, "expect same values")
	assert.Equal(t, string(validData["PASSWORD"]), cred.Password, "expect same values")
}

func TestUnitMonitoringNotEnabled(t *testing.T) {
	instance := testutils.NewPostgresTestDbInstanceCr()
	instance.Spec.Monitoring.Enabled = false
	postgresDbCr := testutils.NewPostgresTestDbCr(instance)
	db, _, _ := dbhelper.FetchDatabaseData(ctx, postgresDbCr, testDbcred, &instance)
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
	instance := testutils.NewPostgresTestDbInstanceCr()
	instance.Spec.Monitoring.Enabled = true
	postgresDbCr := testutils.NewPostgresTestDbCr(instance)

	db, _, _ := dbhelper.FetchDatabaseData(ctx, postgresDbCr, testDbcred, &instance)
	postgresInterface, _ := db.(database.Postgres)

	assert.Equal(t, postgresInterface.Monitoring, true, "expected monitoring is true in postgres interface")
}

func TestUnitPsqlTemplatedSecretGeneratationWithProxy(t *testing.T) {
	instance := testutils.NewPostgresTestDbInstanceCr()
	postgresDbCr := testutils.NewPostgresTestDbCr(instance)
	postgresDbCr.Status.ProxyStatus.Status = true
	postgresDbCr.Spec.SecretsTemplates = map[string]string{
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

	db, _, _ := dbhelper.FetchDatabaseData(ctx, postgresDbCr, testDbcred, &instance)
	connString, err := templates.GenerateTemplatedSecrets(ctx, postgresDbCr, testDbcred, db.GetDatabaseAddress(ctx))
	if err != nil {
		t.Logf("Unexpected error: %s", err)
		t.Fail()
	}
	assert.Equal(t, expectedData, connString, "generated connections string is wrong")
}

func TestUnitPsqlCustomSecretGeneratation(t *testing.T) {
	instance := testutils.NewPostgresTestDbInstanceCr()
	postgresDbCr := testutils.NewPostgresTestDbCr(instance)

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

	db, _, _ := dbhelper.FetchDatabaseData(ctx, postgresDbCr, testDbcred, &instance)
	templatedSecrets, err := templates.GenerateTemplatedSecrets(ctx, postgresDbCr, testDbcred, db.GetDatabaseAddress(ctx))
	if err != nil {
		t.Logf("unexpected error: %s", err)
		t.Fail()
	}
	assert.Equal(t, templatedSecrets, expectedData, "generated connections string is wrong")
}

func TestUnitWrongTemplatedSecretGeneratation(t *testing.T) {
	instance := testutils.NewPostgresTestDbInstanceCr()
	postgresDbCr := testutils.NewPostgresTestDbCr(instance)

	postgresDbCr.Spec.SecretsTemplates = map[string]string{
		"TMPL": "{{ .Protocol }}://{{ .User }}:{{ .Password }}@{{ .DatabaseHost }}:{{ .DatabasePort }}/{{ .DatabaseName }}",
	}

	db, _, _ := dbhelper.FetchDatabaseData(ctx, postgresDbCr, testDbcred, &instance)
	_, err := templates.GenerateTemplatedSecrets(ctx, postgresDbCr, testDbcred, db.GetDatabaseAddress(ctx))
	errSubstr := "can't evaluate field User in type templates.SecretsTemplatesFields"

	assert.Contains(t, err.Error(), errSubstr, "the error doesn't contain expected substring")
}

func TestUnitBlockedTempatedKeysGeneratation(t *testing.T) {
	instance := testutils.NewPostgresTestDbInstanceCr()
	postgresDbCr := testutils.NewPostgresTestDbCr(instance)

	postgresDbCr.Spec.SecretsTemplates = map[string]string{}
	untemplatedFields := []string{templates.FieldMysqlDB, templates.FieldMysqlPassword, templates.FieldMysqlUser, templates.FieldPostgresDB, templates.FieldPostgresUser, templates.FieldPostgressPassword}
	for _, key := range untemplatedFields {
		postgresDbCr.Spec.SecretsTemplates[key] = "DUMMY"
	}
	postgresDbCr.Spec.SecretsTemplates["TMPL"] = "DUMMY"
	expectedData := map[string][]byte{
		"TMPL": []byte("DUMMY"),
	}
	db, _, _ := dbhelper.FetchDatabaseData(ctx, postgresDbCr, testDbcred, &instance)
	sercretData, err := templates.GenerateTemplatedSecrets(ctx, postgresDbCr, testDbcred, db.GetDatabaseAddress(ctx))
	if err != nil {
		t.Logf("unexpected error: %s", err)
		t.Fail()
	}

	dummySecret := v1.Secret{
		Data: map[string][]byte{},
	}

	newSecret := templates.AppendTemplatedSecretData(ctx, postgresDbCr, dummySecret.Data, sercretData)
	assert.Equal(t, newSecret, expectedData, "generated connections string is wrong")
}

func TestUnitObsoleteFieldsRemoving(t *testing.T) {
	instance := testutils.NewPostgresTestDbInstanceCr()
	postgresDbCr := testutils.NewPostgresTestDbCr(instance)

	postgresDbCr.Spec.SecretsTemplates = map[string]string{}
	postgresDbCr.Spec.SecretsTemplates["TMPL"] = "DUMMY"
	expectedData := map[string][]byte{
		"TMPL": []byte("DUMMY"),
	}

	db, _, _ := dbhelper.FetchDatabaseData(ctx, postgresDbCr, testDbcred, &instance)
	secretData, err := templates.GenerateTemplatedSecrets(ctx, postgresDbCr, testDbcred, db.GetDatabaseAddress(ctx))
	if err != nil {
		t.Logf("unexpected error: %s", err)
		t.Fail()
	}
	untemplatedFields := []string{templates.FieldMysqlDB, templates.FieldMysqlPassword, templates.FieldMysqlUser, templates.FieldPostgresDB, templates.FieldPostgresUser, templates.FieldPostgressPassword}
	for _, key := range untemplatedFields {
		secretData[key] = []byte("DUMMY")
	}

	dummySecret := v1.Secret{
		Data: map[string][]byte{
			"TO_REMOVE": []byte("this is supposed to be removed"),
		},
	}
	newSecret := templates.AppendTemplatedSecretData(ctx, postgresDbCr, dummySecret.Data, secretData)
	newSecret = templates.RemoveObsoleteSecret(ctx, postgresDbCr, dummySecret.Data, secretData)

	assert.Equal(t, newSecret, expectedData, "generated connections string is wrong")
}

func TestUnitGetGenericSSLModePostgres(t *testing.T) {
	instance := testutils.NewPostgresTestDbInstanceCr()
	posgresDbCR := testutils.NewPostgresTestDbCr(instance)
	instance.Spec.SSLConnection.Enabled = false
	instance.Spec.SSLConnection.SkipVerify = false
	mode, err := dbhelper.GetGenericSSLMode(posgresDbCR, &instance)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, consts.SSL_DISABLED, mode)

	instance.Spec.SSLConnection.SkipVerify = true
	mode, err = dbhelper.GetGenericSSLMode(posgresDbCR, &instance)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, consts.SSL_DISABLED, mode)

	instance.Spec.SSLConnection.Enabled = true
	instance.Spec.SSLConnection.SkipVerify = true
	mode, err = dbhelper.GetGenericSSLMode(posgresDbCR, &instance)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, consts.SSL_REQUIRED, mode)

	instance.Spec.SSLConnection.SkipVerify = false
	mode, err = dbhelper.GetGenericSSLMode(posgresDbCR, &instance)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, consts.SSL_VERIFY_CA, mode)
}

func TestUnitGetGenericSSLModeMysql(t *testing.T) {
	mysqlDbCR := testutils.NewMysqlTestDbCr()
	instance := testutils.NewMysqlTestDbInstanceCr()

	instance.Spec.SSLConnection.Enabled = false
	instance.Spec.SSLConnection.SkipVerify = false
	mode, err := dbhelper.GetGenericSSLMode(mysqlDbCR, &instance)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, consts.SSL_DISABLED, mode)

	instance.Spec.SSLConnection.SkipVerify = true
	mode, err = dbhelper.GetGenericSSLMode(mysqlDbCR, &instance)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, consts.SSL_DISABLED, mode)

	instance.Spec.SSLConnection.Enabled = true
	instance.Spec.SSLConnection.SkipVerify = true
	mode, err = dbhelper.GetGenericSSLMode(mysqlDbCR, &instance)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, consts.SSL_REQUIRED, mode)

	instance.Spec.SSLConnection.SkipVerify = false
	mode, err = dbhelper.GetGenericSSLMode(mysqlDbCR, &instance)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, consts.SSL_VERIFY_CA, mode)
}

func TestUnitGetSSLModePostgres(t *testing.T) {
	instance := testutils.NewPostgresTestDbInstanceCr()
	posgresDbCR := testutils.NewPostgresTestDbCr(instance)

	instance.Spec.SSLConnection.Enabled = false
	instance.Spec.SSLConnection.SkipVerify = false
	mode, err := dbhelper.GetSSLMode(posgresDbCR, &instance)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "disable", mode)

	instance.Spec.SSLConnection.SkipVerify = true
	mode, err = dbhelper.GetSSLMode(posgresDbCR, &instance)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "disable", mode)

	instance.Spec.SSLConnection.Enabled = true
	instance.Spec.SSLConnection.SkipVerify = true
	mode, err = dbhelper.GetSSLMode(posgresDbCR, &instance)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "require", mode)

	instance.Spec.SSLConnection.SkipVerify = false
	mode, err = dbhelper.GetSSLMode(posgresDbCR, &instance)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "verify-ca", mode)
}

func TestUnitGetSSLModeMysql(t *testing.T) {
	mysqlDbCR := testutils.NewMysqlTestDbCr()
	instance := testutils.NewMysqlTestDbInstanceCr()

	instance.Spec.SSLConnection.Enabled = false
	instance.Spec.SSLConnection.SkipVerify = false
	mode, err := dbhelper.GetSSLMode(mysqlDbCR, &instance)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "disabled", mode)

	instance.Spec.SSLConnection.SkipVerify = true
	mode, err = dbhelper.GetSSLMode(mysqlDbCR, &instance)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "disabled", mode)

	instance.Spec.SSLConnection.Enabled = true
	instance.Spec.SSLConnection.SkipVerify = true
	mode, err = dbhelper.GetSSLMode(mysqlDbCR, &instance)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "required", mode)

	instance.Spec.SSLConnection.SkipVerify = false
	mode, err = dbhelper.GetSSLMode(mysqlDbCR, &instance)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "verify_ca", mode)
}

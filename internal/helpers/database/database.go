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
	"context"
	"errors"
	"fmt"
	"strconv"

	kindav1beta1 "github.com/db-operator/db-operator/v2/api/v1beta1"
	"github.com/db-operator/db-operator/v2/pkg/consts"
	"github.com/db-operator/db-operator/v2/pkg/utils/database"
	"github.com/db-operator/db-operator/v2/pkg/utils/kci"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func FetchDatabaseData(ctx context.Context, dbcr *kindav1beta1.Database, dbCred database.Credentials, instance *kindav1beta1.DbInstance) (database.Database, *database.DatabaseUser, error) {
	log := log.FromContext(ctx)
	host := instance.Status.Info["DB_CONN"]
	port, err := strconv.ParseUint(instance.Status.Info["DB_PORT"], 10, 16)
	if err != nil {
		log.Error(err, "can't get port information from the instanceRef")
		return nil, nil, err
	}

	backend, err := instance.GetBackendType()
	if err != nil {
		log.Error(err, "could not get backend type")
		return nil, nil, err
	}

	monitoringEnabled := instance.IsMonitoringEnabled()

	dbuser := &database.DatabaseUser{
		Username: dbCred.Username,
		Password: dbCred.Password,
		// By default it's set to true, should be overridden
		// according to CR only in the dbuser controller
		GrantToAdmin: true,
	}

	enableRdsIamImpersonate := false
	val, ok := instance.Annotations[consts.RDS_IAM_IMPERSONATE_WORKAROUND]
	if ok {
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			log.Info(
				"can't parse a value of an annotation into a bool, ignoring",
				"annotation", consts.RDS_IAM_IMPERSONATE_WORKAROUND,
				"value", val,
				"error", err,
			)
		} else {
			enableRdsIamImpersonate = boolVal
		}
	}

	switch dbcr.Status.Engine {
	case "postgres":
		// Is force delete enabled
		enableforceDelete := false
		val, ok := dbcr.Annotations[consts.POSTGRES_FORCE_DELETE_DB]
		if ok {
			enableforceDelete, err = strconv.ParseBool(val)
			if err != nil {
				enableforceDelete = false
			}
		}

		extList := dbcr.Spec.Postgres.Extensions
		db := database.Postgres{
			Backend:                     backend,
			Host:                        host,
			Port:                        uint16(port),
			Database:                    dbCred.Name,
			Monitoring:                  monitoringEnabled,
			Extensions:                  extList,
			SSLEnabled:                  instance.Spec.SSLConnection.Enabled,
			SkipCAVerify:                instance.Spec.SSLConnection.SkipVerify,
			DropPublicSchema:            dbcr.Spec.Postgres.DropPublicSchema,
			Schemas:                     dbcr.Spec.Postgres.Schemas,
			Template:                    dbcr.Spec.Postgres.Template,
			MainUser:                    dbuser,
			RDSIAMImpersonateWorkaround: enableRdsIamImpersonate,
			ForceDelete:                 enableforceDelete,
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

func ParseDatabaseSecretData(dbcr *kindav1beta1.Database, data map[string][]byte) (database.Credentials, error) {
	cred := database.Credentials{}

	switch dbcr.Status.Engine {
	case "postgres":
		if name, ok := data[consts.POSTGRES_DB]; ok {
			cred.Name = string(name)
		} else {
			return cred, errors.New("POSTGRES_DB key does not exist in secret data")
		}

		if user, ok := data[consts.POSTGRES_USER]; ok {
			cred.Username = string(user)
		} else {
			return cred, errors.New("POSTGRES_USER key does not exist in secret data")
		}

		if pass, ok := data[consts.POSTGRES_PASSWORD]; ok {
			cred.Password = string(pass)
		} else {
			return cred, errors.New("POSTGRES_PASSWORD key does not exist in secret data")
		}

		return cred, nil
	case "mysql":
		if name, ok := data[consts.MYSQL_DB]; ok {
			cred.Name = string(name)
		} else {
			return cred, errors.New("DB key does not exist in secret data")
		}

		if user, ok := data[consts.MYSQL_USER]; ok {
			cred.Username = string(user)
		} else {
			return cred, errors.New("USER key does not exist in secret data")
		}

		if pass, ok := data[consts.MYSQL_PASSWORD]; ok {
			cred.Password = string(pass)
		} else {
			return cred, errors.New("PASSWORD key does not exist in secret data")
		}

		return cred, nil
	default:
		return cred, errors.New("not supported engine type")
	}
}

// If dbName is empty, it will be generated, that should be used for database resources.
// In case this function is called by dbuser controller, dbName should be taken from the
// `Spec.DatabaseRef` field, so it will ba passed as the last argument
func GenerateDatabaseSecretData(objectMeta metav1.ObjectMeta, engine, dbName, existingUser string) (map[string][]byte, error) {
	const (
		// https://dev.mysql.com/doc/refman/5.7/en/identifier-length.html
		mysqlDBNameLengthLimit = 63
		// https://dev.mysql.com/doc/refman/5.7/en/replication-features-user-names.html
		mysqlUserLengthLimit = 32
	)
	if len(dbName) == 0 {
		dbName = objectMeta.Namespace + "-" + objectMeta.Name
	}
	var dbUser string
	var dbPassword string
	if len(existingUser) > 0 {
		dbUser = existingUser
		dbPassword = ""
	} else {
		var err error
		dbPassword, err = kci.GeneratePass()
		if err != nil {
			return nil, err
		}
		dbUser = objectMeta.Namespace + "-" + objectMeta.Name
	}
	switch engine {
	case "postgres":
		data := map[string][]byte{
			consts.POSTGRES_DB:       []byte(dbName),
			consts.POSTGRES_USER:     []byte(dbUser),
			consts.POSTGRES_PASSWORD: []byte(dbPassword),
		}
		return data, nil
	case "mysql":
		data := map[string][]byte{
			consts.MYSQL_DB:       []byte(kci.StringSanitize(dbName, mysqlDBNameLengthLimit)),
			consts.MYSQL_USER:     []byte(kci.StringSanitize(dbUser, mysqlUserLengthLimit)),
			consts.MYSQL_PASSWORD: []byte(dbPassword),
		}
		return data, nil
	default:
		return nil, errors.New("not supported engine type")
	}
}

func GetSSLMode(dbcr *kindav1beta1.Database, instance *kindav1beta1.DbInstance) (string, error) {
	genericSSL, err := GetGenericSSLMode(dbcr, instance)
	if err != nil {
		return "", err
	}

	if dbcr.Status.Engine == "postgres" {
		switch genericSSL {
		case consts.SSL_DISABLED:
			return "disable", nil
		case consts.SSL_REQUIRED:
			return "require", nil
		case consts.SSL_VERIFY_CA:
			return "verify-ca", nil
		}
	}

	if dbcr.Status.Engine == "mysql" {
		switch genericSSL {
		case consts.SSL_DISABLED:
			return "disabled", nil
		case consts.SSL_REQUIRED:
			return "required", nil
		case consts.SSL_VERIFY_CA:
			return "verify_ca", nil
		}
	}

	return "", fmt.Errorf("unknown database engine: %s", dbcr.Status.Engine)
}

func GetGenericSSLMode(dbcr *kindav1beta1.Database, instance *kindav1beta1.DbInstance) (string, error) {
	if !instance.Spec.SSLConnection.Enabled {
		return consts.SSL_DISABLED, nil
	} else {
		if instance.Spec.SSLConnection.SkipVerify {
			return consts.SSL_REQUIRED, nil
		} else {
			return consts.SSL_VERIFY_CA, nil
		}
	}
}

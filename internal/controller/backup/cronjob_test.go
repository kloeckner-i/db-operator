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

package backup

import (
	"fmt"
	"testing"

	v1 "k8s.io/api/core/v1"

	kindav1beta1 "github.com/db-operator/db-operator/v2/api/v1beta1"
	"github.com/db-operator/db-operator/v2/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestUnitBackupCronGeneric(t *testing.T) {
	dbcr := &kindav1beta1.Database{}
	dbcr.Namespace = "TestNS"
	dbcr.Name = "TestDB"
	instance := &kindav1beta1.DbInstance{}
	instance.Status.Info = map[string]string{"DB_CONN": "TestConnection", "DB_PORT": "1234"}
	instance.Spec.Generic = &kindav1beta1.GenericInstance{BackupHost: "replica.test"}
	instance.Spec.Backup.Bucket = "test-bucket"
	dbcr.Spec.Instance = "staging"
	dbcr.Spec.Backup.Cron = "* * * * *"

	conf, _ := config.LoadConfig("./test/backup_config.yaml")

	instance.Spec.Engine = "postgres"
	funcCronObject, err := BackupCronJobManifest(conf, dbcr, instance)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "postgresbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	instance.Spec.Engine = "mysql"
	funcCronObject, err = BackupCronJobManifest(conf, dbcr, instance)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "mysqlbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	assert.Equal(t, "TestNS", funcCronObject.Namespace)
	assert.Equal(t, "TestDB-backup", funcCronObject.Name)
	assert.Equal(t, "* * * * *", funcCronObject.Spec.Schedule)
	assert.Equal(t, len(funcCronObject.OwnerReferences), 0, "Unexpected size of an OwnerReference")
}

func TestUnitBackupCronGenericEnvFrom(t *testing.T) {
	dbcr := &kindav1beta1.Database{}
	dbcr.Namespace = "TestNS"
	dbcr.Name = "TestDB"
	instance := &kindav1beta1.DbInstance{}
	instance.Status.Info = map[string]string{"DB_CONN": "TestConnection", "DB_PORT": "1234"}
	instance.Spec.Generic = &kindav1beta1.GenericInstance{BackupHost: "replica.test"}
	instance.Spec.Backup.Bucket = "test-bucket"
	dbcr.Spec.Instance = "staging"
	dbcr.Spec.Backup.Cron = "* * * * *"
	dbcr.Spec.Backup.EnvFromSecret = "test-creds"
	conf, _ := config.LoadConfig("./test/backup_config.yaml")

	instance.Spec.Engine = "postgres"
	funcCronObject, err := BackupCronJobManifest(conf, dbcr, instance)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "postgresbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	instance.Spec.Engine = "mysql"
	funcCronObject, err = BackupCronJobManifest(conf, dbcr, instance)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "mysqlbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	assert.Equal(t, "TestNS", funcCronObject.Namespace)
	assert.Equal(t, "TestDB-backup", funcCronObject.Name)
	assert.Equal(t, "* * * * *", funcCronObject.Spec.Schedule)
	assert.Equal(t, len(funcCronObject.OwnerReferences), 0, "Unexpected size of an OwnerReference")
	assert.Equal(t, funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.Name, "test-creds")
}

func TestUnitBackupCronGenericBucket(t *testing.T) {
	dbcr := &kindav1beta1.Database{}
	dbcr.Namespace = "TestNS"
	dbcr.Name = "TestDB"
	instance := &kindav1beta1.DbInstance{}
	instance.Status.Info = map[string]string{"DB_CONN": "TestConnection", "DB_PORT": "1234"}
	instance.Spec.Generic = &kindav1beta1.GenericInstance{BackupHost: "slave.test"}
	instance.Spec.Backup.Bucket = "test-bucket"
	dbcr.Spec.Instance = "staging"
	dbcr.Spec.Backup.Cron = "* * * * *"
	conf, _ := config.LoadConfig("./test/backup_config.yaml")

	instance.Spec.Engine = "postgres"
	funcCronObject, err := BackupCronJobManifest(conf, dbcr, instance)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "postgresbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	instance.Spec.Engine = "mysql"
	funcCronObject, err = BackupCronJobManifest(conf, dbcr, instance)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "mysqlbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	assert.Equal(t, "TestNS", funcCronObject.Namespace)
	assert.Equal(t, "TestDB-backup", funcCronObject.Name)
	assert.Equal(t, "* * * * *", funcCronObject.Spec.Schedule)
	assert.Equal(t, len(funcCronObject.OwnerReferences), 0, "Unexpected size of an OwnerReference")
	assert.Contains(t, funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env, v1.EnvVar{Name: "STORAGE_BUCKET", Value: "test-bucket"})
}

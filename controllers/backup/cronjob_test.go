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
	"os"
	"testing"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/api/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestGCSBackupCronGsql(t *testing.T) {
	dbcr := &kciv1alpha1.Database{}
	dbcr.Namespace = "TestNS"
	dbcr.Name = "TestDB"
	instance := &kciv1alpha1.DbInstance{}
	instance.Status.Info = map[string]string{"DB_CONN": "TestConnection", "DB_PORT": "1234"}
	instance.Spec.Google = &kciv1alpha1.GoogleInstance{InstanceName: "google-instance-1"}
	dbcr.Status.InstanceRef = instance
	dbcr.Spec.Instance = "staging"
	dbcr.Spec.Backup.Cron = "* * * * *"

	os.Setenv("CONFIG_PATH", "./test/backup_config.yaml")
	conf := config.LoadConfig()

	instance.Spec.Engine = "postgres"
	funcCronObject, err := GCSBackupCron(&conf, dbcr)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "postgresbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	instance.Spec.Engine = "mysql"
	funcCronObject, err = GCSBackupCron(&conf, dbcr)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "mysqlbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	assert.Equal(t, "TestNS", funcCronObject.Namespace)
	assert.Equal(t, "TestNS-TestDB-backup", funcCronObject.Name)
	assert.Equal(t, "* * * * *", funcCronObject.Spec.Schedule)
}

func TestGCSBackupCronGeneric(t *testing.T) {
	dbcr := &kciv1alpha1.Database{}
	dbcr.Namespace = "TestNS"
	dbcr.Name = "TestDB"
	instance := &kciv1alpha1.DbInstance{}
	instance.Status.Info = map[string]string{"DB_CONN": "TestConnection", "DB_PORT": "1234"}
	instance.Spec.Generic = &kciv1alpha1.GenericInstance{BackupHost: "slave.test"}
	dbcr.Status.InstanceRef = instance
	dbcr.Spec.Instance = "staging"
	dbcr.Spec.Backup.Cron = "* * * * *"

	os.Setenv("CONFIG_PATH", "./test/backup_config.yaml")
	conf := config.LoadConfig()

	instance.Spec.Engine = "postgres"
	funcCronObject, err := GCSBackupCron(&conf, dbcr)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "postgresbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	instance.Spec.Engine = "mysql"
	funcCronObject, err = GCSBackupCron(&conf, dbcr)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "mysqlbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	assert.Equal(t, "TestNS", funcCronObject.Namespace)
	assert.Equal(t, "TestNS-TestDB-backup", funcCronObject.Name)
	assert.Equal(t, "* * * * *", funcCronObject.Spec.Schedule)
}

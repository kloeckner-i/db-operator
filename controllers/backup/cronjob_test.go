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

	kciv1beta1 "github.com/kloeckner-i/db-operator/api/v1beta1"
	"github.com/kloeckner-i/db-operator/pkg/config"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGCSBackupCronGsql(t *testing.T) {
	ownership := []metav1.OwnerReference{}
	dbcr := &kciv1beta1.Database{}
	dbcr.Namespace = "TestNS"
	dbcr.Name = "TestDB"
	instance := &kciv1beta1.DbInstance{}
	instance.Status.Info = map[string]string{"DB_CONN": "TestConnection", "DB_PORT": "1234"}
	instance.Spec.Google = &kciv1beta1.GoogleInstance{InstanceName: "google-instance-1"}
	dbcr.Status.InstanceRef = instance
	dbcr.Spec.Instance = "staging"
	dbcr.Spec.Backup.Cron = "* * * * *"

	os.Setenv("CONFIG_PATH", "./test/backup_config.yaml")
	conf := config.LoadConfig()

	instance.Spec.Engine = "postgres"
	funcCronObject, err := GCSBackupCron(&conf, dbcr, ownership)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "postgresbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	instance.Spec.Engine = "mysql"
	funcCronObject, err = GCSBackupCron(&conf, dbcr, ownership)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "mysqlbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	assert.Equal(t, "TestNS", funcCronObject.Namespace)
	assert.Equal(t, "TestNS-TestDB-backup", funcCronObject.Name)
	assert.Equal(t, "* * * * *", funcCronObject.Spec.Schedule)
}

func TestGCSBackupCronGeneric(t *testing.T) {
	ownership := []metav1.OwnerReference{}
	dbcr := &kciv1beta1.Database{}
	dbcr.Namespace = "TestNS"
	dbcr.Name = "TestDB"
	instance := &kciv1beta1.DbInstance{}
	instance.Status.Info = map[string]string{"DB_CONN": "TestConnection", "DB_PORT": "1234"}
	instance.Spec.Generic = &kciv1beta1.GenericInstance{BackupHost: "slave.test"}
	dbcr.Status.InstanceRef = instance
	dbcr.Spec.Instance = "staging"
	dbcr.Spec.Backup.Cron = "* * * * *"

	os.Setenv("CONFIG_PATH", "./test/backup_config.yaml")
	conf := config.LoadConfig()

	instance.Spec.Engine = "postgres"
	funcCronObject, err := GCSBackupCron(&conf, dbcr, ownership)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "postgresbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	instance.Spec.Engine = "mysql"
	funcCronObject, err = GCSBackupCron(&conf, dbcr, ownership)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "mysqlbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	assert.Equal(t, "TestNS", funcCronObject.Namespace)
	assert.Equal(t, "TestNS-TestDB-backup", funcCronObject.Name)
	assert.Equal(t, "* * * * *", funcCronObject.Spec.Schedule)
	assert.Equal(t, len(funcCronObject.OwnerReferences), 0, "Unexpected size of an OwnerReference")
}

func TestGCSBackupCronGenericWithOwnerReference(t *testing.T) {
	ownership := []metav1.OwnerReference{}
	ownership = append(ownership, metav1.OwnerReference{
		APIVersion: "api-version",
		Kind:       "kind",
		Name:       "name",
		UID:        "uid",
	})
	dbcr := &kciv1beta1.Database{}
	dbcr.Namespace = "TestNS"
	dbcr.Name = "TestDB"
	instance := &kciv1beta1.DbInstance{}
	instance.Status.Info = map[string]string{"DB_CONN": "TestConnection", "DB_PORT": "1234"}
	instance.Spec.Generic = &kciv1beta1.GenericInstance{BackupHost: "slave.test"}
	dbcr.Status.InstanceRef = instance
	dbcr.Spec.Instance = "staging"
	dbcr.Spec.Backup.Cron = "* * * * *"

	os.Setenv("CONFIG_PATH", "./test/backup_config.yaml")
	conf := config.LoadConfig()

	instance.Spec.Engine = "postgres"
	funcCronObject, err := GCSBackupCron(&conf, dbcr, ownership)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, len(funcCronObject.OwnerReferences), 1, "Unexpected size of an OwnerReference")
	assert.Equal(t, funcCronObject.OwnerReferences[0].APIVersion, ownership[0].APIVersion, "API Version in the OwnerReference is wrong")
	assert.Equal(t, funcCronObject.OwnerReferences[0].Kind, ownership[0].Kind, "Kind in the OwnerReference is wrong")
	assert.Equal(t, funcCronObject.OwnerReferences[0].Name, ownership[0].Name, "Name in the OwnerReference is wrong")
	assert.Equal(t, funcCronObject.OwnerReferences[0].UID, ownership[0].UID, "UID in the OwnerReference is wrong")
}
func TestGetResourceRequirements(t *testing.T) {
	os.Setenv("CONFIG_PATH", "./test/backup_config.yaml")
	conf := config.LoadConfig()

	expected := v1.ResourceRequirements{
		Requests: map[v1.ResourceName]resource.Quantity{v1.ResourceCPU: resource.MustParse("50m"), v1.ResourceMemory: resource.MustParse("50Mi")},
		Limits:   map[v1.ResourceName]resource.Quantity{v1.ResourceCPU: resource.MustParse("100m"), v1.ResourceMemory: resource.MustParse("100Mi")},
	}
	result := getResourceRequirements(&conf)
	assert.Equal(t, expected, result)
}

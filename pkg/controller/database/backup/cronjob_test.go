package backup

import (
	"fmt"
	"os"
	"testing"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
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
	conf = config.LoadConfig()

	instance.Spec.Engine = "postgres"
	funcCronObject, err := GCSBackupCron(dbcr)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "postgresbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	instance.Spec.Engine = "mysql"
	funcCronObject, err = GCSBackupCron(dbcr)
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
	conf = config.LoadConfig()

	instance.Spec.Engine = "postgres"
	funcCronObject, err := GCSBackupCron(dbcr)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName)
	assert.Equal(t, "postgresbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	instance.Spec.Engine = "mysql"
	funcCronObject, err = GCSBackupCron(dbcr)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "mysqlbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	assert.Equal(t, "TestNS", funcCronObject.Namespace)
	assert.Equal(t, "TestNS-TestDB-backup", funcCronObject.Name)
	assert.Equal(t, "* * * * *", funcCronObject.Spec.Schedule)
}

func TestGCSBackupCronAmazonServiceAccountFromConfig(t *testing.T) {
	dbcr := &kciv1alpha1.Database{}
	dbcr.Namespace = "TestNS"
	dbcr.Name = "TestDB"
	instance := &kciv1alpha1.DbInstance{}
	instance.Status.Info = map[string]string{"DB_CONN": "TestConnection", "DB_PORT": "1234"}
	instance.Spec.Amazon = &kciv1alpha1.AmazonInstance{BackupHost: "slave.test"}

	dbcr.Status.InstanceRef = instance
	dbcr.Spec.Instance = "staging"
	dbcr.Spec.Backup.Cron = "* * * * *"

	os.Setenv("CONFIG_PATH", "./test/backup_config.yaml")
	conf = config.LoadConfig()

	instance.Spec.Engine = "postgres"
	funcCronObject, err := GCSBackupCron(dbcr)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "backup", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName)
	assert.Equal(t, "postgresbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	instance.Spec.Engine = "mysql"
	funcCronObject, err = GCSBackupCron(dbcr)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "mysqlbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	assert.Equal(t, "backup", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName)
	assert.Equal(t, "TestNS", funcCronObject.Namespace)
	assert.Equal(t, "TestNS-TestDB-backup", funcCronObject.Name)
	assert.Equal(t, "* * * * *", funcCronObject.Spec.Schedule)
}

func TestGCSBackupCronAmazonServiceAccountFromInstance(t *testing.T) {
	dbcr := &kciv1alpha1.Database{}
	dbcr.Namespace = "TestNS"
	dbcr.Name = "TestDB"
	instance := &kciv1alpha1.DbInstance{}
	instance.Status.Info = map[string]string{"DB_CONN": "TestConnection", "DB_PORT": "1234"}
	instance.Spec.Amazon = &kciv1alpha1.AmazonInstance{BackupHost: "slave.test", ServiceAccountName: "backup01"}

	dbcr.Status.InstanceRef = instance
	dbcr.Spec.Instance = "staging"
	dbcr.Spec.Backup.Cron = "* * * * *"

	os.Setenv("CONFIG_PATH", "./test/backup_config.yaml")
	conf = config.LoadConfig()

	instance.Spec.Engine = "postgres"
	funcCronObject, err := GCSBackupCron(dbcr)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "backup01", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName)
	assert.Equal(t, "postgresbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	instance.Spec.Engine = "mysql"
	funcCronObject, err = GCSBackupCron(dbcr)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "mysqlbackupimage:latest", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image)

	assert.Equal(t, "backup01", funcCronObject.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName)
	assert.Equal(t, "TestNS", funcCronObject.Namespace)
	assert.Equal(t, "TestNS-TestDB-backup", funcCronObject.Name)
	assert.Equal(t, "* * * * *", funcCronObject.Spec.Schedule)
}

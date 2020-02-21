package backup

import (
	"fmt"
	"os"
	"testing"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"

	"github.com/stretchr/testify/assert"
)

func TestGCSBackupCron(t *testing.T) {
	dbcr := &kciv1alpha1.Database{}
	dbcr.Namespace = "TestNS"
	dbcr.Name = "TestDB"
	instance := &kciv1alpha1.DbInstance{}
	instance.Status.Info = map[string]string{"DB_CONN": "TestConnection"}
	dbcr.Status.InstanceRef = instance

	dbcr.Spec.Instance = "staging"
	dbcr.Spec.Backup.Cron = "* * * * *"

	os.Setenv("CONFIG_PATH", "../../config/test/config_ok.yaml")
	funcCronObject, err := GCSBackupCron(dbcr)
	if err != nil {
		fmt.Print(err)
	}

	assert.Equal(t, "TestNS", funcCronObject.Namespace)
	assert.Equal(t, "TestNS-TestDB-backup", funcCronObject.Name)
	assert.Equal(t, "* * * * *", funcCronObject.Spec.Schedule)
}

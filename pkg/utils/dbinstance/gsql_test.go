package dbinstance

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/option"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
)

func mockGetSqladminService(ctx context.Context) (*sqladmin.Service, error) {
	opts := []option.ClientOption{
		option.WithEndpoint("http://127.0.0.1:8080"),
	}

	sqladminService, err := sqladmin.NewService(ctx, opts...)
	if err != nil {
		logrus.Debugf("error occurs during getting sqladminService %s", err)
		return nil, err
	}

	return sqladminService, nil
}

func (ins *Gsql) mockWaitUntilRunnable() error {
	const delay = 3
	time.Sleep(delay * time.Second)

	err := kci.Retry(10, 2*time.Second, func() error {
		instance, err := ins.getInstance()
		if err != nil {
			return err
		}
		logrus.Debugf("waiting gsql instance %s state: %s", ins.Name, instance.State)

		if instance.State != "RUNNABLE" {
			return errors.New("gsql instance not ready yet")
		}

		return nil
	})

	if err != nil {
		instance, err := ins.getInstance()
		if err != nil {
			return err
		}

		return fmt.Errorf("gsql instance state not ready %s", instance.State)
	}

	return nil
}

func mockGsqlConfig() string {
	return `{
		"databaseVersion": "POSTGRES_12",
		"settings": {
		  "tier": "db-f1-micro",
		  "availabilityType": "ZONAL",
		  "pricingPlan": "PER_USE",
		  "replicationType": "SYNCHRONOUS",
		  "activationPolicy": "ALWAYS",
		  "ipConfiguration": {
			"authorizedNetworks": [],
			"ipv4Enabled": true
		  },
		  "dataDiskType": "PD_SSD",
		  "backupConfiguration": {
			"enabled": false
		  },
		  "storageAutoResizeLimit": "0",
		  "storageAutoResize": true
		},
		"backendType": "SECOND_GEN",
		"region": "europe-west1"
}`
}

func TestGsqlGetInstance(t *testing.T) {
	patch := monkey.Patch(getSqladminService, mockGetSqladminService)
	defer patch.Unpatch()

	myGsql := &Gsql{
		Name:      "test-instance",
		ProjectID: "test-project",
	}

	rs, err := myGsql.getInstance()
	logrus.Infof("%#v\n, %s", rs, err)
	assert.NoError(t, err)
}

func TestGsqlCreateInvalidInstance(t *testing.T) {
	patchSqladmin := monkey.Patch(getSqladminService, mockGetSqladminService)
	defer patchSqladmin.Unpatch()

	myGsql := &Gsql{
		Name:      "test-instance",
		ProjectID: "test-project",
	}

	err := myGsql.createInstance()
	assert.Error(t, err)
}

func TestGsqlCreateInstance(t *testing.T) {
	patch := monkey.Patch(getSqladminService, mockGetSqladminService)
	defer patch.Unpatch()

	myGsql := &Gsql{
		Name:      "test-instance",
		ProjectID: "test-project",
		Config:    mockGsqlConfig(),
	}

	patchWait := monkey.Patch((*Gsql).waitUntilRunnable, (*Gsql).mockWaitUntilRunnable)
	defer patchWait.Unpatch()

	err := myGsql.createInstance()
	assert.NoError(t, err)
}

func TestGsqlWaitUntilRunnable(t *testing.T) {
	myGsql := &Gsql{
		Name:      "test-instance",
		ProjectID: "test-project",
	}

	patchWait := monkey.Patch((*Gsql).waitUntilRunnable, (*Gsql).mockWaitUntilRunnable)
	defer patchWait.Unpatch()

	err := myGsql.waitUntilRunnable()
	assert.NoError(t, err)
}

func TestGsqlCreateExistingInstance(t *testing.T) {
	patch := monkey.Patch(getSqladminService, mockGetSqladminService)
	defer patch.Unpatch()

	myGsql := &Gsql{
		Name:      "test-instance",
		ProjectID: "test-project",
		Config:    mockGsqlConfig(),
	}

	err := myGsql.createInstance()
	assert.Error(t, err)
}

func TestGsqlUpdateInstance(t *testing.T) {
	patch := monkey.Patch(getSqladminService, mockGetSqladminService)
	defer patch.Unpatch()

	myGsql := &Gsql{
		Name:      "test-instance",
		ProjectID: "test-project",
		Config:    mockGsqlConfig(),
	}

	patchWait := monkey.Patch((*Gsql).waitUntilRunnable, (*Gsql).mockWaitUntilRunnable)
	defer patchWait.Unpatch()

	err := myGsql.updateInstance()
	assert.NoError(t, err)
}

func TestGsqlUpdateUser(t *testing.T) {
	patch := monkey.Patch(getSqladminService, mockGetSqladminService)
	defer patch.Unpatch()

	myGsql := &Gsql{
		Name:      "test-instance",
		ProjectID: "test-project",
		Config:    mockGsqlConfig(),
		User:      "test-use1r",
		Password:  "testPassw0rd",
	}

	myGsql.updateUser()
}

package dbinstance

import (
	"errors"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// func mockGetSqladminService(ctx context.Context) (*sqladmin.Service, error) {
// 	opts := []option.ClientOption{
// 		option.WithEndpoint("http://127.0.0.1:8080"),
// 	}

// 	sqladminService, err := sqladmin.NewService(ctx, opts...)
// 	if err != nil {
// 		logrus.Debugf("error occurs during getting sqladminService %s", err)
// 		return nil, err
// 	}

// 	return sqladminService, nil
// }

func (ins *Gsql) mockWaitUntilRunnable() error {
	logrus.Debugf("waiting gsql instance %s", ins.Name)

	time.Sleep(10 * time.Second)

	instance, err := ins.getInstance()
	if err != nil {
		return err
	}

	if instance.State != "RUNNABLE" {
		return errors.New("gsql instance not ready yet")
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

func TestGsqlGetInstanceNonExist(t *testing.T) {
	// patch := monkey.Patch(getSqladminService, mockGetSqladminService)
	// defer patch.Unpatch()

	myGsql := &Gsql{
		Name:        "test-instance",
		ProjectID:   "test-project",
		APIEndpoint: "http://127.0.0.1:8080",
	}

	rs, err := myGsql.getInstance()
	logrus.Infof("%#v\n, %s", rs, err)
	assert.Error(t, err)
}

func TestGsqlCreateInvalidInstance(t *testing.T) {
	// patchSqladmin := monkey.Patch(getSqladminService, mockGetSqladminService)
	// defer patchSqladmin.Unpatch()

	myGsql := &Gsql{
		Name:        "test-instance",
		ProjectID:   "test-project",
		APIEndpoint: "http://127.0.0.1:8080",
	}

	err := myGsql.createInstance()
	assert.Error(t, err)
}

func TestGsqlCreateInstance(t *testing.T) {
	// patch := monkey.Patch(getSqladminService, mockGetSqladminService)
	// defer patch.Unpatch()

	myGsql := &Gsql{
		Name:        "test-instance",
		ProjectID:   "test-project",
		APIEndpoint: "http://127.0.0.1:8080",
		Config:      mockGsqlConfig(),
	}

	patchWait := monkey.Patch((*Gsql).waitUntilRunnable, (*Gsql).mockWaitUntilRunnable)
	defer patchWait.Unpatch()

	err := myGsql.createInstance()
	assert.NoError(t, err)
}

func TestGsqlGetInstanceExist(t *testing.T) {
	// patch := monkey.Patch(getSqladminService, mockGetSqladminService)
	// defer patch.Unpatch()

	myGsql := &Gsql{
		Name:        "test-instance",
		ProjectID:   "test-project",
		APIEndpoint: "http://127.0.0.1:8080",
	}

	rs, err := myGsql.getInstance()
	logrus.Infof("%#v\n, %s", rs, err)
	assert.NoError(t, err)
}

func TestGsqlCreateExistingInstance(t *testing.T) {
	// patch := monkey.Patch(getSqladminService, mockGetSqladminService)
	// defer patch.Unpatch()

	myGsql := &Gsql{
		Name:        "test-instance",
		ProjectID:   "test-project",
		APIEndpoint: "http://127.0.0.1:8080",
		Config:      mockGsqlConfig(),
	}

	err := myGsql.createInstance()
	assert.Error(t, err)
}

func TestGsqlUpdateInstance(t *testing.T) {
	// patch := monkey.Patch(getSqladminService, mockGetSqladminService)
	// defer patch.Unpatch()

	myGsql := &Gsql{
		Name:        "test-instance",
		ProjectID:   "test-project",
		APIEndpoint: "http://127.0.0.1:8080",
		Config:      mockGsqlConfig(),
	}

	patchWait := monkey.Patch((*Gsql).waitUntilRunnable, (*Gsql).mockWaitUntilRunnable)
	defer patchWait.Unpatch()

	err := myGsql.updateInstance()
	assert.NoError(t, err)
}

func TestGsqlUpdateUser(t *testing.T) {
	// patch := monkey.Patch(getSqladminService, mockGetSqladminService)
	// defer patch.Unpatch()

	myGsql := &Gsql{
		Name:        "test-instance",
		ProjectID:   "test-project",
		APIEndpoint: "http://127.0.0.1:8080",
		Config:      mockGsqlConfig(),
		User:        "test-use1r",
		Password:    "testPassw0rd",
	}

	myGsql.updateUser()
}

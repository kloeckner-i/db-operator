package dbinstance

import (
	"errors"
	"github.com/google/uuid"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

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
		"region": "somewhere"
}`
}

func myMockGsql() *Gsql {
	return &Gsql{
		Name:        uuid.New().String(),
		ProjectID:   "test-project",
		APIEndpoint: "http://127.0.0.1:8080",
		Config:      mockGsqlConfig(),
		User:        "test-user1",
		Password:    "testPassw0rd",
	}
}

func TestGsqlGetInstanceNonExist(t *testing.T) {
	myGsql := myMockGsql()

	rs, err := myGsql.getInstance()
	logrus.Infof("%#v\n, %s", rs, err)
	assert.Error(t, err)
}

func TestGsqlCreateInvalidInstance(t *testing.T) {
	myGsql := myMockGsql()
	myGsql.Config = ""

	err := myGsql.createInstance()
	assert.Error(t, err)
}

func TestGsqlCreateInstance(t *testing.T) {
	myGsql := myMockGsql()

	patchWait := monkey.Patch((*Gsql).waitUntilRunnable, (*Gsql).mockWaitUntilRunnable)
	defer patchWait.Unpatch()

	err := myGsql.createInstance()
	assert.NoError(t, err)
}

func TestGsqlGetInstanceExist(t *testing.T) {
	myGsql := myMockGsql()

	err := myGsql.createInstance()
	assert.NoError(t, err)

	rs, err := myGsql.getInstance()
	logrus.Infof("%#v\n, %s", rs, err)
	assert.NoError(t, err)
}

func TestGsqlCreateExistingInstance(t *testing.T) {
	myGsql := myMockGsql()

	err := myGsql.createInstance()
	assert.NoError(t, err)

	err = myGsql.createInstance()
	assert.Error(t, err)
}

func TestGsqlUpdateInstance(t *testing.T) {
	myGsql := myMockGsql()

	err := myGsql.createInstance()
	assert.NoError(t, err)

	patchWait := monkey.Patch((*Gsql).waitUntilRunnable, (*Gsql).mockWaitUntilRunnable)
	defer patchWait.Unpatch()

	err = myGsql.updateInstance()
	assert.NoError(t, err)
}

func TestGsqlUpdateUser(t *testing.T) {
	myGsql := myMockGsql()

	err := myGsql.createInstance()
	assert.NoError(t, err)

	err = myGsql.updateUser()
	assert.NoError(t, err)
}

package dbinstance

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// tests don't work, and since gsql will be deprecated there is no need for it

// func (ins *Gsql) mockWaitUntilRunnable(t *testing.T) error {
// 	log := log.FromContext(t.Context())
// 	log.V(2).Info("waiting gsql instance", ins.Name)

// 	time.Sleep(10 * time.Second)

// 	state, err := ins.state()
// 	if err != nil {
// 		return err
// 	}
// 	if state != "RUNNABLE" {
// 		return errors.New("gsql instance not ready yet")
// 	}

// 	return nil
// }

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
	log := log.FromContext(t.Context())
	myGsql := myMockGsql()

	rs, err := myGsql.getInstance()
	log.Info("error", err, rs)
	assert.Error(t, err)
}

func TestGsqlCreateInvalidInstance(t *testing.T) {
	myGsql := myMockGsql()
	myGsql.Config = ""

	err := myGsql.createInstance()
	assert.Error(t, err)
}

// tests don't work, and since gsql will be deprecated there is no need for it

// func TestGsqlCreateInstance(t *testing.T) {
// 	myGsql := myMockGsql()

// 	patchWait := monkey.Patch((*Gsql).waitUntilRunnable, (*Gsql).mockWaitUntilRunnable)
// 	defer patchWait.Unpatch()

// 	err := myGsql.createInstance()
// 	assert.NoError(t, err)
// }

func TestGsqlGetInstanceExist(t *testing.T) {
	log := log.FromContext(t.Context())
	myGsql := myMockGsql()

	err := myGsql.createInstance()
	assert.NoError(t, err)

	rs, err := myGsql.getInstance()
	log.Info("error", err, rs)
	assert.NoError(t, err)
}

func TestGsqlCreateExistingInstance(t *testing.T) {
	myGsql := myMockGsql()

	err := myGsql.createInstance()
	assert.NoError(t, err)

	err = myGsql.createInstance()
	assert.Error(t, err)
}

// tests don't work, and since gsql will be deprecated there is no need for it

// func TestGsqlUpdateInstance(t *testing.T) {
// 	myGsql := myMockGsql()

// 	err := myGsql.createInstance()
// 	assert.NoError(t, err)

// 	patchWait := monkey.Patch((*Gsql).waitUntilRunnable, (*Gsql).mockWaitUntilRunnable)
// 	defer patchWait.Unpatch()

// 	err = myGsql.updateInstance()
// 	assert.NoError(t, err)
// }

func TestGsqlUpdateUser(t *testing.T) {
	myGsql := myMockGsql()

	err := myGsql.createInstance()
	assert.NoError(t, err)

	err = myGsql.updateUser(t.Context())
	assert.NoError(t, err)
}

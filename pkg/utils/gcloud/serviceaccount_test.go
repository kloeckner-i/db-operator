package gcloud

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGetServiceAccount(t *testing.T) {
	// positive test
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./test/serviceaccount.json")
	serviceaccount := GetServiceAccount()
	if serviceaccount.ProjectID != "test-project" {
		t.Errorf("Unexpected %v", serviceaccount.ProjectID)
	}

	// negative test
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./test/fake.json")
	// rollback ExitFunc to default
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	fatalCalled := false
	logrus.StandardLogger().ExitFunc = func(int) { fatalCalled = true }
	expectedFatal := true
	GetServiceAccount()
	assert.Equal(t, expectedFatal, fatalCalled)
}

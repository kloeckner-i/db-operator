package config

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	os.Setenv("CONFIG_PATH", "./test/config_ok.yaml")
	confLoad := LoadConfig()
	confStatic := Config{}

	confStatic.Instances.Google.ClientSecretName = "cloudsql-readonly-serviceaccount"
	assert.Equal(t, confStatic.Instances.Google.ClientSecretName, confLoad.Instances.Google.ClientSecretName, "Values should be match")
	assert.EqualValues(t, confLoad.Backup.ActiveDeadlineSeconds, int64(600))
}

func TestLoadConfigFailCases(t *testing.T) {
	// rollback ExitFunc to default
	defer func() { logrus.StandardLogger().ExitFunc = nil }()
	fatalCalled := false
	logrus.StandardLogger().ExitFunc = func(int) { fatalCalled = true }
	expectedFatal := true
	os.Setenv("CONFIG_PATH", "./test/config_NotFound.yaml")
	LoadConfig()
	assert.Equal(t, expectedFatal, fatalCalled)

	os.Setenv("CONFIG_PATH", "./test/config_Invalid.yaml")
	LoadConfig()
	assert.Equal(t, expectedFatal, fatalCalled)
}

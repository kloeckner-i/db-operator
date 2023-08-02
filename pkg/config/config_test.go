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

package config

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestUnitLoadConfig(t *testing.T) {
	os.Setenv("CONFIG_PATH", "./test/config_ok.yaml")
	confLoad := LoadConfig()
	confStatic := Config{}

	confStatic.Instances.Google.ClientSecretName = "cloudsql-readonly-serviceaccount"
	assert.Equal(t, confStatic.Instances.Google.ClientSecretName, confLoad.Instances.Google.ClientSecretName, "Values should be match")
	assert.EqualValues(t, confLoad.Backup.ActiveDeadlineSeconds, int64(600))
}

func TestUnitLoadConfigFailCases(t *testing.T) {
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

func TestUnitBackupResourceConfig(t *testing.T) {
	os.Setenv("CONFIG_PATH", "./test/config_backup.yaml")
	conf := LoadConfig()
	assert.Equal(t, conf.Backup.Resource.Requests.Cpu, "50m")
	assert.Equal(t, conf.Backup.Resource.Requests.Memory, "50Mi")
	assert.Equal(t, conf.Backup.Resource.Limits.Cpu, "100m")
	assert.Equal(t, conf.Backup.Resource.Limits.Memory, "100Mi")
}

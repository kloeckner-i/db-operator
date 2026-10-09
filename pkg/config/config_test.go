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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnitLoadConfigFailCases(t *testing.T) {
	conf, err := LoadConfig("./test/config_NotFound.yaml")
	assert.Error(t, err)
	assert.Nil(t, conf)

	conf, err = LoadConfig("./test/config_Invalid.yaml")
	assert.Error(t, err)
	assert.Nil(t, conf)
}

func TestUnitBackupResourceConfig(t *testing.T) {
	conf, err := LoadConfig("./test/config_backup.yaml")
	assert.NoError(t, err)
	t.Log(conf)
	assert.Equal(t, conf.Backup.Resources.Requests.Cpu().String(), "50m")
	assert.Equal(t, conf.Backup.Resources.Requests.Memory().String(), "50Mi")
	assert.Equal(t, conf.Backup.Resources.Limits.Cpu().String(), "100m")
	assert.Equal(t, conf.Backup.Resources.Limits.Memory().String(), "100Mi")
}

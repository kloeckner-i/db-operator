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

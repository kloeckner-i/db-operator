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

package dbinstance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMysqlGenericInstanceCreate if creating mysql generic instance works as expected
func TestMysqlGenericInstanceCreate(t *testing.T) {
	mysqlInstance := testGenericMysqlInstance()
	_, err := Create(context.TODO(), mysqlInstance)
	assert.Error(t, err, "expected error already exits %v", err)

	mysqlInstance.Host = "wronghost"
	_, err = Create(context.TODO(), mysqlInstance)
	assert.Error(t, err, "expected error %v", err)
}

// TestPostgresGenericInstanceCreate if creating postgres generic instance works as expected
func TestPostgresGenericInstanceCreate(t *testing.T) {
	postgresInstance := testGenericPostgresInstance()

	_, err := Create(context.TODO(), postgresInstance)
	assert.Error(t, err, "expected error already exits %v", err)

	postgresInstance.Host = "wronghost"
	_, err = Create(context.TODO(), postgresInstance)
	assert.Error(t, err, "expected no error %v", err)
}

// TestMysqlGenericInstanceUpdateNoError if upgrading mysql generic instance works as expected
func TestMysqlGenericInstanceUpdateNoError(t *testing.T) {
	mysqlInstance := testGenericMysqlInstance()
	_, err := Update(context.TODO(), mysqlInstance)
	assert.NoError(t, err, "expected no error %v", err)
}

// TestMysqlGenericInstanceUpdateNonExist checks upgrading non existing mysql generic instance throws error
func TestMysqlGenericInstanceUpdateNonExist(t *testing.T) {
	mysqlInstance := testGenericMysqlInstance()
	mysqlInstance.Host = "wronghost"
	_, err := Update(context.TODO(), mysqlInstance)
	assert.Error(t, err, "expected error %v", err)
}

// TestPostgresGenericInstanceUpdate if upgrading postgres generic instance works as expected
func TestPostgresGenericInstanceUpdate(t *testing.T) {
	postgresInstance := testGenericPostgresInstance()
	_, err := Update(context.TODO(), postgresInstance)
	assert.NoError(t, err, "expected no error %v", err)

	postgresInstance.Host = "wronghost"
	_, err = Update(context.TODO(), postgresInstance)
	assert.Error(t, err, "expected error %v", err)
}

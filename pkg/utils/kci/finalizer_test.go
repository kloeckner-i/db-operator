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

package kci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUnitFinalizers(t *testing.T) {
	md := metav1.ObjectMeta{}
	addString1 := "db.test1"
	addString2 := "db.test2"
	addString3 := "db.test3"

	AddFinalizer(&md, addString1)
	finalizers := md.GetFinalizers()
	assert.Contains(t, finalizers, addString1)

	AddFinalizer(&md, addString2)
	finalizers = md.GetFinalizers()
	assert.Contains(t, finalizers, addString2)

	AddFinalizer(&md, addString3)
	finalizers = md.GetFinalizers()
	assert.Contains(t, finalizers, addString3)

	AddFinalizer(&md, addString3)
	finalizers2 := md.GetFinalizers()
	assert.Contains(t, finalizers2, addString1)

	RemoveFinalizer(&md, addString3)
	finalizers = md.GetFinalizers()
	assert.NotContains(t, finalizers, addString3)
}

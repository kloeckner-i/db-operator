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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddFinalizer adds finalizer to object metadata
func AddFinalizer(om *metav1.ObjectMeta, item string) *metav1.ObjectMeta {
	om.SetFinalizers(appendIfMissing(om.GetFinalizers(), item))
	return om
}

// RemoveFinalizer removes finalizer from object metadata
func RemoveFinalizer(om *metav1.ObjectMeta, item string) *metav1.ObjectMeta {
	om.SetFinalizers(removeItem(om.GetFinalizers(), item))
	return om
}

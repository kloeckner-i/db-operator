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

package controllers

import (
	"context"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/api/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	crdv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func isDBSpecChanged(dbcr *kciv1alpha1.Database) bool {
	annotations := dbcr.ObjectMeta.GetAnnotations()

	return annotations["checksum/spec"] != kci.GenerateChecksum(dbcr.Spec)
}

func addDBSpecChecksum(dbcr *kciv1alpha1.Database) {
	annotations := dbcr.ObjectMeta.GetAnnotations()
	if len(annotations) == 0 {
		annotations = make(map[string]string)
	}

	annotations["checksum/spec"] = kci.GenerateChecksum(dbcr.Spec)
	dbcr.ObjectMeta.SetAnnotations(annotations)
}

func isDBInstanceSpecChanged(ctx context.Context, dbin *kciv1alpha1.DbInstance) bool {
	checksums := dbin.Status.Checksums
	if checksums["spec"] != kci.GenerateChecksum(dbin.Spec) {
		return true
	}

	if backend, _ := dbin.GetBackendType(); backend == "google" {
		instanceConfig, _ := kci.GetConfigResource(ctx, dbin.Spec.Google.ConfigmapName.ToKubernetesType())
		if checksums["config"] != kci.GenerateChecksum(instanceConfig) {
			return true
		}
	}

	return false
}

func addDBInstanceChecksumStatus(ctx context.Context, dbin *kciv1alpha1.DbInstance) {
	checksums := dbin.Status.Checksums
	if len(checksums) == 0 {
		checksums = make(map[string]string)
	}
	checksums["spec"] = kci.GenerateChecksum(dbin.Spec)

	if backend, _ := dbin.GetBackendType(); backend == "google" {
		instanceConfig, _ := kci.GetConfigResource(ctx, dbin.Spec.Google.ConfigmapName.ToKubernetesType())
		checksums["config"] = kci.GenerateChecksum(instanceConfig)
	}

	dbin.Status.Checksums = checksums
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func checkCRD(crds crdv1.CustomResourceDefinitionList, api string) bool {
	for _, crd := range crds.Items {
		if crd.Name == api {
			return true
		}
	}
	return false
}

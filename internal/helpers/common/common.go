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

package common

import (
	"context"
	"reflect"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"

	kindav1beta1 "github.com/db-operator/db-operator/v2/api/v1beta1"
	"github.com/db-operator/db-operator/v2/pkg/consts"
	"github.com/db-operator/db-operator/v2/pkg/utils/kci"
	crdv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var OperatorVersion string

func IsDBChanged(dbcr *kindav1beta1.Database, databaseSecret *corev1.Secret) bool {
	annotations := dbcr.GetAnnotations()

	hash, err := kci.GenerateChecksum(dbcr.Spec)
	// just in case
	if err != nil {
		return true
	}
	return annotations["checksum/spec"] != hash ||
		annotations["checksum/secret"] != GenerateChecksumSecretValue(databaseSecret)
}

func AddDBChecksum(dbcr *kindav1beta1.Database, databaseSecret *corev1.Secret) {
	annotations := dbcr.GetAnnotations()
	if len(annotations) == 0 {
		annotations = make(map[string]string)
	}

	annotations["checksum/spec"], _ = kci.GenerateChecksum(dbcr.Spec)
	annotations["checksum/secret"] = GenerateChecksumSecretValue(databaseSecret)
	dbcr.SetAnnotations(annotations)
}

func GenerateChecksumSecretValue(databaseSecret *corev1.Secret) string {
	if databaseSecret == nil || databaseSecret.Data == nil {
		return ""
	}

	// Not all the fields must be stored in the checksum, because templates are appending new entries, that should not trigger a new recocniliation
	relevantKeys := []string{consts.POSTGRES_DB, consts.POSTGRES_PASSWORD, consts.POSTGRES_USER, consts.MYSQL_DB, consts.MYSQL_PASSWORD, consts.MYSQL_USER}
	relevantData := map[string][]byte{}
	for key, value := range databaseSecret.Data {
		if slices.Contains(relevantKeys, key) {
			relevantData[key] = value
		}
	}

	hash, err := kci.GenerateChecksum(relevantData)
	if err != nil {
		return ""
	}
	return hash
}

// DbInstanceData contains all externally referenced objects of a DatabaseInstance
type DbInstanceData struct {
	AdminSecret  *corev1.Secret
	ConfigMap    *corev1.ConfigMap
	ClientSecret *corev1.Secret
	HostFrom     client.Object
	PortFrom     client.Object
	PublicIPFrom client.Object
}

func IsDBInstanceChanged(ctx context.Context, dbin *kindav1beta1.DbInstance, data DbInstanceData) bool {
	currentChecksums := GenerateDBInstanceChecksums(dbin, data)
	return !reflect.DeepEqual(currentChecksums, dbin.Status.Checksums)
}

// GenerateDBInstanceChecksums serves as the single source of truth for calculating the checksums
// of all objects a DBInstance depends on (spec, secrets, and configmaps).
func GenerateDBInstanceChecksums(dbin *kindav1beta1.DbInstance, data DbInstanceData) map[string]string {
	checksums := make(map[string]string)

	hash, err := kci.GenerateChecksum(dbin.Spec)
	// just to ensure the state
	if err == nil {
		checksums["spec"] = hash
	}

	backend, _ := dbin.GetBackendType()
	if backend == "google" {
		if data.ConfigMap != nil {
			hash, _ := kci.GenerateChecksum(data.ConfigMap.Data)
			checksums["config"] = hash
		}
		if dbin.Spec.Google.ClientSecret.Name != "" && data.ClientSecret != nil {
			hash, _ := kci.GenerateChecksum(data.ClientSecret.Data)
			checksums["clientSecret"] = hash
		}
	}

	if backend, _ := dbin.GetBackendType(); backend == "generic" {
		if from := dbin.Spec.Generic.HostFrom; from != nil && data.HostFrom != nil {
			checksums["hostFrom"] = getObjectDataChecksum(data.HostFrom)
		}
		if from := dbin.Spec.Generic.PortFrom; from != nil && data.PortFrom != nil {
			checksums["portFrom"] = getObjectDataChecksum(data.PortFrom)
		}
		if from := dbin.Spec.Generic.PublicIPFrom; from != nil && data.PublicIPFrom != nil {
			checksums["publicIPFrom"] = getObjectDataChecksum(data.PublicIPFrom)
		}
	}

	if data.AdminSecret != nil {
		hash, _ := kci.GenerateChecksum(data.AdminSecret.Data)
		checksums["adminSecret"] = hash
	}

	return checksums
}

func getObjectDataChecksum(obj client.Object) string {
	switch o := obj.(type) {
	case *corev1.Secret:
		hash, _ := kci.GenerateChecksum(o.Data)
		return hash
	case *corev1.ConfigMap:
		hash, _ := kci.GenerateChecksum(o.Data)
		return hash
	}
	return ""
}

func EnsureLabel(ctx context.Context, c client.Client, obj client.Object, key string, value string) error {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	// If the label is already set, no need to trigger an update
	if labels[key] == value {
		return nil
	}

	labels[key] = value
	obj.SetLabels(labels)
	return c.Update(ctx, obj)
}

func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func SliceContainsSubString(slice []string, s string) bool {
	for _, item := range slice {
		if strings.Contains(item, s) {
			return true
		}
	}
	return false
}

// InCrdList returns true if monitoring is enabled in DbInstance spec.
func InCrdList(crds crdv1.CustomResourceDefinitionList, api string) bool {
	for _, crd := range crds.Items {
		if crd.Name == api {
			return true
		}
	}
	return false
}

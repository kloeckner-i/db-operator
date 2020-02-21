package dbinstance

import (
	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
)

func isChanged(dbin *kciv1alpha1.DbInstance) bool {
	checksums := dbin.Status.Checksums
	if checksums["spec"] != kci.GenerateChecksum(dbin.Spec) {
		return true
	}

	if backend, _ := dbin.GetBackendType(); backend == "google" {
		instanceConfig, _ := kci.GetConfigResource(dbin.Spec.Google.ConfigmapName)
		if checksums["config"] != kci.GenerateChecksum(instanceConfig) {
			return true
		}
	}

	return false
}

func addChecksumStatus(dbin *kciv1alpha1.DbInstance) {
	checksums := dbin.Status.Checksums
	if len(checksums) == 0 {
		checksums = make(map[string]string)
	}
	checksums["spec"] = kci.GenerateChecksum(dbin.Spec)

	if backend, _ := dbin.GetBackendType(); backend == "google" {
		instanceConfig, _ := kci.GetConfigResource(dbin.Spec.Google.ConfigmapName)
		checksums["config"] = kci.GenerateChecksum(instanceConfig)
	}

	dbin.Status.Checksums = checksums
}

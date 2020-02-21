package database

import (
	"crypto/md5"
	"fmt"
	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	"io"
)

func isSpecChanged(dbcr *kciv1alpha1.Database) bool {
	annotations := dbcr.ObjectMeta.GetAnnotations()
	if annotations["checksum/spec"] != kci.GenerateChecksum(dbcr.Spec) {
		return true
	}
	return false
}

func addSpecChecksum(dbcr *kciv1alpha1.Database) {
	annotations := dbcr.ObjectMeta.GetAnnotations()
	if len(annotations) == 0 {
		annotations = make(map[string]string)
	}

	annotations["checksum/spec"] = kci.GenerateChecksum(dbcr.Spec)
	dbcr.ObjectMeta.SetAnnotations(annotations)
}

func stringShortner(s string) string {
	h := md5.New()
	io.WriteString(h, s)
	str := fmt.Sprintf("%x", h.Sum(nil))
	return str[0:16]
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

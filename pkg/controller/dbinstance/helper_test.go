package dbinstance

import (
	"errors"
	"testing"

	"bou.ke/monkey"
	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func mockGetOperatorNamespace() (string, error) {
	return "testns", nil
}

func testConfigmap1(nsName types.NamespacedName) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	cm.Namespace = nsName.Namespace
	cm.Name = nsName.Name

	data := make(map[string]string)
	data["config"] = "test1"
	cm.Data = data

	return cm, nil
}

func testConfigmap2(nsName types.NamespacedName) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	cm.Namespace = nsName.Namespace
	cm.Name = nsName.Name

	data := make(map[string]string)
	data["config"] = "test2"
	cm.Data = data

	return cm, nil
}

func errorConfigmap(namespace, configmapName string) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	return cm, errors.New("whatever error")
}

func testAdminSecret(namespace, secretName string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}

	data := make(map[string][]byte)
	data["user"] = []byte("user")
	data["password"] = []byte("securepassword")

	secret.Data = data
	return secret, nil
}

func TestSpecChanged(t *testing.T) {
	dbin := &kciv1alpha1.DbInstance{}
	before := kciv1alpha1.DbInstanceSpec{
		AdminUserSecret: types.NamespacedName{
			Namespace: "test",
			Name:      "secret1",
		},
	}
	dbin.Spec = before
	addChecksumStatus(dbin)
	nochange := isChanged(dbin)
	assert.Equal(t, nochange, false, "expected false")

	after := kciv1alpha1.DbInstanceSpec{
		AdminUserSecret: types.NamespacedName{
			Namespace: "test",
			Name:      "secret2",
		},
	}
	dbin.Spec = after
	change := isChanged(dbin)
	assert.Equal(t, change, true, "expected true")
}

func TestConfigChanged(t *testing.T) {
	dbin := &kciv1alpha1.DbInstance{}
	dbin.Spec.Google = &kciv1alpha1.GoogleInstance{
		InstanceName: "test",
		ConfigmapName: types.NamespacedName{
			Namespace: "testNS",
			Name:      "test",
		},
	}

	patch := monkey.Patch(kci.GetConfigResource, testConfigmap1)
	defer patch.Unpatch()
	addChecksumStatus(dbin)

	nochange := isChanged(dbin)
	assert.Equal(t, nochange, false, "expected false")

	patch = monkey.Patch(kci.GetConfigResource, testConfigmap2)
	change := isChanged(dbin)
	assert.Equal(t, change, true, "expected true")
}

func TestAddChecksumStatus(t *testing.T) {
	dbin := &kciv1alpha1.DbInstance{}
	addChecksumStatus(dbin)
	checksums := dbin.Status.Checksums
	assert.NotEqual(t, checksums, map[string]string{}, "annotation should have checksum")
}

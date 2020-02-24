package database

import (
	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/test"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newPostgresTestDbCr() *kciv1alpha1.Database {
	o := metav1.ObjectMeta{Namespace: "TestNS"}
	s := kciv1alpha1.DatabaseSpec{SecretName: "TestSec"}

	info := make(map[string]string)
	info["DB_PORT"] = "5432"

	db := kciv1alpha1.Database{
		ObjectMeta: o,
		Spec:       s,
		Status: kciv1alpha1.DatabaseStatus{
			InstanceRef: &kciv1alpha1.DbInstance{
				Spec: kciv1alpha1.DbInstanceSpec{
					Engine: "postgres",
					Generic: &kciv1alpha1.GenericInstance{
						Host: test.GetPostgresHost(),
						Port: test.GetPostgresPort(),
					},
				},
				Status: kciv1alpha1.DbInstanceStatus{Info: info},
			},
		},
	}

	return &db
}

func newMysqlTestDbCr() *kciv1alpha1.Database {
	o := metav1.ObjectMeta{Namespace: "TestNS"}
	s := kciv1alpha1.DatabaseSpec{SecretName: "TestSec"}

	info := make(map[string]string)
	info["DB_PORT"] = "3306"

	db := kciv1alpha1.Database{
		ObjectMeta: o,
		Spec:       s,
		Status: kciv1alpha1.DatabaseStatus{
			InstanceRef: &kciv1alpha1.DbInstance{
				Spec: kciv1alpha1.DbInstanceSpec{
					Engine: "mysql",
					Generic: &kciv1alpha1.GenericInstance{
						Host: test.GetMysqlHost(),
						Port: test.GetMysqlPort(),
					},
				},
				Status: kciv1alpha1.DbInstanceStatus{Info: info},
			},
		},
	}

	return &db
}

func TestIsSpecChanged(t *testing.T) {
	db := newPostgresTestDbCr()
	addSpecChecksum(db)
	nochange := isSpecChanged(db)
	assert.Equal(t, nochange, false, "expected false")

	db.Spec.SecretName = "NewSec"
	change := isSpecChanged(db)
	assert.Equal(t, change, true, "expected true")
}

func TestStringShortner(t *testing.T) {
	assert.Equal(t, "e08ba29f56785332", stringShortner("short_string"))
	assert.Equal(t, "8f54fbbc6fefae00", stringShortner("short_string_longer"))
	assert.Equal(t, "f32225e6124f2fab", stringShortner("short_string_to_long_to_see_all_chars_in_result"))
}

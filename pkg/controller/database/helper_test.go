package database

import (
	"testing"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/test"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newPostgresTestDbInstanceCr() kciv1alpha1.DbInstance {
	info := make(map[string]string)
	info["DB_PORT"] = "5432"

	return kciv1alpha1.DbInstance{
		Spec: kciv1alpha1.DbInstanceSpec{
			Engine: "postgres",
			DbInstanceSource: kciv1alpha1.DbInstanceSource{
				Generic: &kciv1alpha1.GenericInstance{
					Host: test.GetPostgresHost(),
					Port: test.GetPostgresPort(),
				},
			},
		},
		Status: kciv1alpha1.DbInstanceStatus{Info: info},
	}
}

func newPostgresTestDbCr(instanceRef kciv1alpha1.DbInstance) *kciv1alpha1.Database {
	o := metav1.ObjectMeta{Namespace: "TestNS"}
	s := kciv1alpha1.DatabaseSpec{SecretName: "TestSec"}

	db := kciv1alpha1.Database{
		ObjectMeta: o,
		Spec:       s,
		Status: kciv1alpha1.DatabaseStatus{
			InstanceRef: &instanceRef,
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
					DbInstanceSource: kciv1alpha1.DbInstanceSource{
						Generic: &kciv1alpha1.GenericInstance{
							Host: test.GetMysqlHost(),
							Port: test.GetMysqlPort(),
						},
					},
				},
				Status: kciv1alpha1.DbInstanceStatus{Info: info},
			},
		},
	}

	return &db
}

func TestIsSpecChanged(t *testing.T) {
	db := newPostgresTestDbCr(newPostgresTestDbInstanceCr())
	addSpecChecksum(db)
	nochange := isSpecChanged(db)
	assert.Equal(t, nochange, false, "expected false")

	db.Spec.SecretName = "NewSec"
	change := isSpecChanged(db)
	assert.Equal(t, change, true, "expected true")
}

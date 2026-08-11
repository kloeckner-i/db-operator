package v1beta1_test

import (
	"testing"

	"github.com/db-operator/db-operator/v2/api/v1beta1"
	"github.com/db-operator/db-operator/v2/pkg/types"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var dbin types.KindaObject

func TestKindObjectDbToClientObject(t *testing.T) {
	dbin = &v1beta1.Database{
		TypeMeta: v1.TypeMeta{
			Kind:       "Database",
			APIVersion: "v1beta1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "database",
			Namespace: "default",
		},
		Spec: v1beta1.DatabaseSpec{},
	}

	dbObj, ok := dbin.(client.Object)
	assert.True(t, ok)

	assert.Equal(t, dbObj.GetName(), "database")
	assert.Equal(t, dbObj.GetNamespace(), "default")
}

func TestKindObjectDbUserToClientObject(t *testing.T) {
	dbin = &v1beta1.DbUser{
		TypeMeta: v1.TypeMeta{
			Kind:       "DbUser",
			APIVersion: "v1beta1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "dbuser",
			Namespace: "default",
		},
		Spec: v1beta1.DbUserSpec{},
	}

	dbObj, ok := dbin.(client.Object)
	assert.True(t, ok)

	assert.Equal(t, dbObj.GetName(), "dbuser")
	assert.Equal(t, dbObj.GetNamespace(), "default")
}

func TestKindObjectDbinToClientObject(t *testing.T) {
	dbin = &v1beta1.DbInstance{
		TypeMeta: v1.TypeMeta{
			Kind:       "DbInstance",
			APIVersion: "v1beta1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "dbinstance",
			Namespace: "default",
		},
		Spec: v1beta1.DbInstanceSpec{},
	}

	dbObj, ok := dbin.(client.Object)
	assert.True(t, ok)

	assert.Equal(t, dbObj.GetName(), "dbinstance")
	assert.Equal(t, dbObj.GetNamespace(), "default")
}

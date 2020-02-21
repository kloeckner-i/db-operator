package kci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFinalizers(t *testing.T) {
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

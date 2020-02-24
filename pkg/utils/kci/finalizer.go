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

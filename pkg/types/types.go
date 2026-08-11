package types

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// An interface that implements common memthods for all kinda objects
type KindaObject interface {
	IsCleanup() bool
	IsDeleted() bool
	GetSecretName() string
	// client.Object default methods to avoid casting later
	GetName() string
	GetNamespace() string
	GetObjectKind() schema.ObjectKind
	GetUID() types.UID
}

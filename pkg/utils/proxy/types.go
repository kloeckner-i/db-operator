package proxy

import (
	v1apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

// Proxy for database
type Proxy interface {
	buildDeployment() (*v1apps.Deployment, error)
	buildService() (*v1.Service, error)
	buildConfigMap() (v1.ConfigMap, error)
}

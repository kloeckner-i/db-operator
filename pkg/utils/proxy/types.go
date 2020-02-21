package proxy

import (
	v1apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

// CloudProxy for google sql instance
type CloudProxy struct {
	NamePrefix             string
	Namespace              string
	InstanceConnectionName string
	AccessSecretName       string
	Engine                 string
	Port                   int32
	Labels                 map[string]string
}

// Proxy for database
type Proxy interface {
	buildDeployment() (*v1apps.Deployment, error)
	buildService() (*v1.Service, error)
}

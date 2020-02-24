package proxy

import (
	"github.com/sirupsen/logrus"
	v1apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

// BuildDeployment builds kubernetes deployment object to create proxy container of the database
func BuildDeployment(proxy Proxy) (*v1apps.Deployment, error) {
	deploy, err := proxy.buildDeployment()
	if err != nil {
		logrus.Error("failed building cloudsql proxy deployment")
		return nil, err
	}

	return deploy, nil
}

// BuildService builds kubernetes service object for proxy service of the database
func BuildService(proxy Proxy) (*v1.Service, error) {
	svc, err := proxy.buildService()
	if err != nil {
		logrus.Error("failed building cloudsql proxy service")
		return nil, err
	}

	return svc, nil
}

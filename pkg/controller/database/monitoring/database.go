package monitoring

import (
	"fmt"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"

	v1 "k8s.io/api/core/v1"
	v1apps "k8s.io/api/apps/v1"
)

// Deployment builds kubernetes deployment object
// to run prometheus exporter to expose dbcr metrics
func Deployment(dbcr *kciv1alpha1.Database) (*v1apps.Deployment, error) {
	engine, err := dbcr.GetEngineType()
	if err != nil {
		return nil, err
	}

	switch engine {
	case "postgres":
		return pgDeployment(dbcr)
	default:
		return nil, fmt.Errorf("Monitoring: db engine monitoring for %s not implemented", engine)
	}
}

// ConfigMap builds kubernetes configmap object
// which contains query to execute by prometheus exporter
func ConfigMap(dbcr *kciv1alpha1.Database) (*v1.ConfigMap, error) {
	engine, err := dbcr.GetEngineType()
	if err != nil {
		return nil, err
	}

	switch engine {
	case "postgres":
		return pgExporterQueryCM(dbcr)
	default:
		return nil, fmt.Errorf("monitoring: exporter query creation for db engine %s not implemented", engine)
	}

}

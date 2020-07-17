package monitoring

import (
	"testing"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	v1apps "k8s.io/api/apps/v1"
)

func testPGDeployment(t *testing.T) {
	dbcr := &kciv1alpha1.Database{}

	// mock PGDeployment
	pgDeploymentMock := func(*kciv1alpha1.Database) (*v1apps.Deployment, error) {
		return &v1apps.Deployment{}, nil
	}
	patch := monkey.Patch(pgDeployment, pgDeploymentMock)
	defer patch.Unpatch()

	res, err := pgDeployment(dbcr)
	assert.Equal(t, &v1apps.Deployment{}, res)
	assert.Equal(t, nil, err)
}

func testPGExporterQueryCM(t *testing.T) {
	dbcr := &kciv1alpha1.Database{}

	// mock PGDeployment
	pgExporterQueryCMMock := func(*kciv1alpha1.Database) (*v1.ConfigMap, error) {
		return &v1.ConfigMap{}, nil
	}
	patch := monkey.Patch(pgExporterQueryCM, pgExporterQueryCMMock)
	defer patch.Unpatch()

	res, err := pgExporterQueryCM(dbcr)
	assert.Equal(t, &v1.ConfigMap{}, res)
	assert.Equal(t, nil, err)
}

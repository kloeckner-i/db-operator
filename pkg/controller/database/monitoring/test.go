package monitoring

import (
	"testing"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
)

func testPGDeployment(t *testing.T) {
	dbcr := &kciv1alpha1.Database{}

	// mock PGDeployment
	pgDeploymentMock := func(*kciv1alpha1.Database) (*extensionsv1beta1.Deployment, error) {
		return &extensionsv1beta1.Deployment{}, nil
	}
	patch := monkey.Patch(pgDeployment, pgDeploymentMock)
	defer patch.Unpatch()

	res, err := pgDeployment(dbcr)
	assert.Equal(t, &extensionsv1beta1.Deployment{}, res)
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

package monitoring

import (
	"fmt"
	"github.com/kloeckner-i/db-operator/pkg/config"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	v1 "k8s.io/api/core/v1"
)

func pgExporterQueryCM(conf *config.Config, dbcr *kciv1alpha1.Database) (*v1.ConfigMap, error) {
	cmData := make(map[string]string)
	cmData["queries.yaml"] = conf.Monitoring.Postgres.Queries
	return kci.ConfigMapBuilder(queryCMName(dbcr), dbcr.GetNamespace(), cmData), nil
}

func queryCMName(dbcr *kciv1alpha1.Database) string {
	return fmt.Sprintf("%s-monitoring-query", dbcr.Name)
}

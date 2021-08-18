/*
 * Copyright 2021 kloeckner.i GmbH
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package monitoring

import (
	"fmt"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/api/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/config"
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

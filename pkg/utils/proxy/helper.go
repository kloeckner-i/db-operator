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

package proxy

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func podAntiAffinity(labelSelector map[string]string) *v1.PodAntiAffinity {
	var weight int32 = 1
	return &v1.PodAntiAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
			{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: labelSelector,
				},
				TopologyKey: "kubernetes.io/hostname",
			},
		},
		PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
			{
				PodAffinityTerm: v1.PodAffinityTerm{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: labelSelector,
					},
					TopologyKey: "failure-domain.beta.kubernetes.io/zone",
				},
				Weight: weight,
			},
		},
	}
}

// IsMonitoringEnabled get status of MonitoringEnabled
func IsMonitoringEnabled(proxy Proxy) bool {
	return proxy.isMonitoringEnabled()
}

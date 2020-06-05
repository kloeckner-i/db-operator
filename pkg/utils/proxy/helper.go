package proxy

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func podAntiAffinity(labelSelector map[string]string) *v1.PodAntiAffinity {
	var weight int32 = 1
	return &v1.PodAntiAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
			v1.PodAffinityTerm{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: labelSelector,
				},
				TopologyKey: "kubernetes.io/hostname",
			},
		},
		PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
			v1.WeightedPodAffinityTerm{
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

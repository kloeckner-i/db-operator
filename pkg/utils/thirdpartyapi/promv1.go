package thirdpartyapi

import (
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	crdv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func AppendToScheme(scheme *runtime.Scheme) {

	scheme.AddKnownTypes(crdv1.SchemeGroupVersion, &crdv1.CustomResourceDefinitionList{}, &crdv1.CustomResourceDefinition{})
	metav1.AddToGroupVersion(scheme, crdv1.SchemeGroupVersion)

	scheme.AddKnownTypes(promv1.SchemeGroupVersion, &promv1.PodMonitor{})
	scheme.AddKnownTypes(promv1.SchemeGroupVersion, &promv1.ServiceMonitor{})
	metav1.AddToGroupVersion(scheme, promv1.SchemeGroupVersion)

}

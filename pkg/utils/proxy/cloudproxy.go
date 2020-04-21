package proxy

import (
	"fmt"
	"strconv"

	"github.com/kloeckner-i/db-operator/pkg/config"

	v1apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var conf = config.Config{}

const instanceAccessSecretVolumeName string = "gcloud-secret"

func (cp *CloudProxy) buildService() (*v1.Service, error) {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cp.NamePrefix + "-svc",
			Namespace: cp.Namespace,
			Labels:    cp.Labels,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Name:     cp.Engine,
					Protocol: v1.ProtocolTCP,
					Port:     cp.Port,
				},
			},
			Selector: cp.Labels,
		},
	}, nil
}

func (cp *CloudProxy) buildDeployment() (*v1apps.Deployment, error) {
	spec, err := deploymentSpec(cp.InstanceConnectionName, cp.Port, cp.Labels, cp.AccessSecretName)
	if err != nil {
		return nil, err
	}

	return &v1apps.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "extensions/apps",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cp.NamePrefix + "-cloudproxy",
			Namespace: cp.Namespace,
			Labels:    cp.Labels,
		},
		Spec: spec,
	}, nil

}

func deploymentSpec(conn string, port int32, labels map[string]string, instanceAccessSecret string) (v1apps.DeploymentSpec, error) {
	var replicas int32 = 2

	container, err := container(conn, port)
	if err != nil {
		return v1apps.DeploymentSpec{}, err
	}

	volumes := buildVolumes(instanceAccessSecret)

	return v1apps.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: v1.PodSpec{
				Containers:    []v1.Container{container},
				NodeSelector:  conf.Instances.Google.ProxyConfig.NodeSelector,
				RestartPolicy: v1.RestartPolicyAlways,
				Volumes:       volumes,
				Affinity: &v1.Affinity{
					PodAntiAffinity: podAntiAffinity(labels),
				},
			},
		},
	}, nil
}

func container(conn string, port int32) (v1.Container, error) {
	RunAsUser := int64(2)
	AllowPrivilegeEscalation := false
	instanceArg := fmt.Sprintf("-instances=%s=tcp:0.0.0.0:%s", conn, strconv.FormatInt(int64(port), 10))

	return v1.Container{
		Name:    "cloudsql-proxy",
		Image:   conf.Instances.Google.ProxyConfig.Image,
		Command: []string{"/cloud_sql_proxy"},
		Args:    []string{instanceArg, "-credential_file=/srv/gcloud/credentials.json"},
		SecurityContext: &v1.SecurityContext{
			RunAsUser:                &RunAsUser,
			AllowPrivilegeEscalation: &AllowPrivilegeEscalation,
		},
		ImagePullPolicy: v1.PullIfNotPresent,
		Ports: []v1.ContainerPort{
			v1.ContainerPort{
				Name:          "sqlport",
				ContainerPort: port,
				Protocol:      v1.ProtocolTCP,
			},
		},
		VolumeMounts: []v1.VolumeMount{
			v1.VolumeMount{
				Name:      instanceAccessSecretVolumeName,
				MountPath: "/srv/gcloud/",
			},
		},
	}, nil
}

func buildVolumes(instanceAccessSecretName string) []v1.Volume {
	return []v1.Volume{
		v1.Volume{
			Name: instanceAccessSecretVolumeName,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: instanceAccessSecretName,
				},
			},
		},
	}
}

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

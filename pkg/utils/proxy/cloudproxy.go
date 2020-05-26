package proxy

import (
	"fmt"
	"strconv"

	"github.com/kloeckner-i/db-operator/pkg/config"

	v1apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CloudProxy for google sql instance
type CloudProxy struct {
	NamePrefix             string
	Namespace              string
	InstanceConnectionName string
	AccessSecretName       string
	Engine                 string
	Port                   int32
	Labels                 map[string]string
}

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
	spec, err := cp.deploymentSpec()
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

func (cp *CloudProxy) deploymentSpec() (v1apps.DeploymentSpec, error) {
	var replicas int32 = 2

	container, err := cp.container()
	if err != nil {
		return v1apps.DeploymentSpec{}, err
	}

	volumes := []v1.Volume{
		v1.Volume{
			Name: instanceAccessSecretVolumeName,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: cp.AccessSecretName,
				},
			},
		},
	}

	return v1apps.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: cp.Labels,
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: cp.Labels,
			},
			Spec: v1.PodSpec{
				Containers:    []v1.Container{container},
				NodeSelector:  conf.Instances.Google.ProxyConfig.NodeSelector,
				RestartPolicy: v1.RestartPolicyAlways,
				Volumes:       volumes,
				Affinity: &v1.Affinity{
					PodAntiAffinity: podAntiAffinity(cp.Labels),
				},
			},
		},
	}, nil
}

func (cp *CloudProxy) container() (v1.Container, error) {
	RunAsUser := int64(2)
	AllowPrivilegeEscalation := false
	instanceArg := fmt.Sprintf("-instances=%s=tcp:0.0.0.0:%s", cp.InstanceConnectionName, strconv.FormatInt(int64(cp.Port), 10))

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
				ContainerPort: cp.Port,
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

func (cp *CloudProxy) buildConfigMap() (v1.ConfigMap, error) {
	return v1.ConfigMap{}, nil
}

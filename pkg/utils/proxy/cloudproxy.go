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
	"fmt"
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
	Conf                   *config.Config
}

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
			APIVersion: "apps/v1",
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

	terminationGracePeriodSeconds := int64(120) // force kill pod after this time

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
				NodeSelector:  cp.Conf.Instances.Google.ProxyConfig.NodeSelector,
				RestartPolicy: v1.RestartPolicyAlways,
				Volumes:       volumes,
				Affinity: &v1.Affinity{
					PodAntiAffinity: podAntiAffinity(cp.Labels),
				},
				TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
			},
		},
	}, nil
}

func (cp *CloudProxy) container() (v1.Container, error) {
	RunAsUser := int64(2)
	AllowPrivilegeEscalation := false
	listenArg := fmt.Sprintf("--listen=0.0.0.0:%d", cp.Port)
	instanceArg := fmt.Sprintf("--instance=%s", cp.InstanceConnectionName)

	return v1.Container{
		Name:    "db-auth-gateway",
		Image:   cp.Conf.Instances.Google.ProxyConfig.Image,
		Command: []string{"/usr/local/bin/db-auth-gateway"},
		Args:    []string{"--credential-file=/srv/gcloud/credentials.json", listenArg, instanceArg},
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

func (cp *CloudProxy) buildConfigMap() (*v1.ConfigMap, error) {
	return nil, nil
}

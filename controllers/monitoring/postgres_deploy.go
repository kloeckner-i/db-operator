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
	"github.com/kloeckner-i/db-operator/pkg/config"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/api/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"

	"github.com/sirupsen/logrus"
	v1apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func pgDeployment(conf *config.Config, dbcr *kciv1alpha1.Database) (*v1apps.Deployment, error) {
	return &v1apps.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbcr.Name + "-" + "pgexporter",
			Namespace: dbcr.Namespace,
			Labels:    kci.BaseLabelBuilder(),
		},
		Spec: pgDeploymentSpec(conf, dbcr),
	}, nil
}

func pgDeploymentSpec(conf *config.Config, dbcr *kciv1alpha1.Database) v1apps.DeploymentSpec {
	Replicas := int32(1)

	return v1apps.DeploymentSpec{
		Replicas: &Replicas,
		Strategy: v1apps.DeploymentStrategy{
			Type: v1apps.RecreateDeploymentStrategyType,
		},
		Selector: &metav1.LabelSelector{
			MatchLabels: pgPodLabels(),
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: pgPodLabels(),
				Annotations: map[string]string{
					"prometheus.io/port":   "60000",
					"prometheus.io/scrape": "true",
				},
			},
			Spec: pgPodSpec(conf, dbcr),
		},
	}
}

func pgPodSpec(conf *config.Config, dbcr *kciv1alpha1.Database) v1.PodSpec {
	return v1.PodSpec{
		Volumes: pgVolumes(dbcr),
		Containers: []v1.Container{
			pgContainerExporter(conf, dbcr),
		},
		NodeSelector:  conf.Monitoring.NodeSelector,
		RestartPolicy: v1.RestartPolicyAlways,
	}
}

func pgPodLabels() map[string]string {
	labels := map[string]string{
		"app": "pgexporter",
	}

	return kci.LabelBuilder(labels)
}

func pgContainerExporterResources() v1.ResourceRequirements {
	return v1.ResourceRequirements{
		Limits: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("100m"),
			v1.ResourceMemory: resource.MustParse("256Mi"),
		},
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse("50m"),
			v1.ResourceMemory: resource.MustParse("64Mi"),
		},
	}
}

func pgContainerExporter(conf *config.Config, dbcr *kciv1alpha1.Database) v1.Container {
	RunAsUser := int64(2)
	AllowPrivilegeEscalation := false

	return v1.Container{
		Name:  "exporter",
		Image: conf.Monitoring.Postgres.ExporterImage,
		SecurityContext: &v1.SecurityContext{
			RunAsUser:                &RunAsUser,
			AllowPrivilegeEscalation: &AllowPrivilegeEscalation,
		},
		ImagePullPolicy: v1.PullAlways,
		VolumeMounts:    pgVolumeMountsExporter(),
		Env:             pgEnvExporter(dbcr),
	}
}

func pgVolumeMountsExporter() []v1.VolumeMount {
	return []v1.VolumeMount{
		v1.VolumeMount{
			Name:      "db-secrets",
			MountPath: "/run/secrets/db-secrets",
		},
		v1.VolumeMount{
			Name:      "queries",
			MountPath: "/run/cm/queries/queries.yaml",
			SubPath:   "queries.yaml",
		},
	}
}

func pgVolumes(dbcr *kciv1alpha1.Database) []v1.Volume {
	return []v1.Volume{
		v1.Volume{
			Name: "db-secrets",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: dbcr.Spec.SecretName,
				},
			},
		},
		v1.Volume{
			Name: "queries",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: queryCMName(dbcr),
					},
				},
			},
		},
	}
}

func pgEnvExporter(dbcr *kciv1alpha1.Database) []v1.EnvVar {
	host := "db-" + dbcr.Name + "-svc"
	instance, err := dbcr.GetInstanceRef()
	if err != nil {
		logrus.Fatal(err)
	}
	port := instance.Status.Info["DB_PORT"]

	return []v1.EnvVar{
		v1.EnvVar{
			Name: "DATA_SOURCE_URI", Value: host + ":" + port + "/postgres?sslmode=disable",
		},
		v1.EnvVar{
			Name: "DATA_SOURCE_PASS_FILE", Value: "/run/secrets/db-secrets/POSTGRES_PASSWORD",
		},
		v1.EnvVar{
			Name: "DATA_SOURCE_USER_FILE", Value: "/run/secrets/db-secrets/POSTGRES_USER",
		},
		v1.EnvVar{
			Name: "PG_EXPORTER_WEB_LISTEN_ADDRESS", Value: ":60000",
		},
		v1.EnvVar{
			Name: "PG_EXPORTER_EXTEND_QUERY_PATH", Value: "/run/cm/queries/queries.yaml",
		},
		v1.EnvVar{
			Name: "PG_EXPORTER_CONSTANT_LABELS", Value: fmt.Sprintf("dbinstance=%s", dbcr.Spec.Instance),
		},
		v1.EnvVar{
			Name: "PG_EXPORTER_DISABLE_DEFAULT_METRICS", Value: "true",
		},
		v1.EnvVar{
			Name: "PG_EXPORTER_DISABLE_SETTINGS_METRICS", Value: "true",
		},
	}
}

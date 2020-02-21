package monitoring

import (
	"fmt"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/config"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var conf = config.Config{}

func pgDeployment(dbcr *kciv1alpha1.Database) (*extensionsv1beta1.Deployment, error) {
	return &extensionsv1beta1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbcr.Name + "-" + "pgexporter",
			Namespace: dbcr.Namespace,
			Labels:    kci.BaseLabelBuilder(),
		},
		Spec: pgDeploymentSpec(dbcr),
	}, nil
}

func pgDeploymentSpec(dbcr *kciv1alpha1.Database) extensionsv1beta1.DeploymentSpec {
	Replicas := int32(1)

	return extensionsv1beta1.DeploymentSpec{
		Replicas: &Replicas,
		Strategy: extensionsv1beta1.DeploymentStrategy{
			Type: extensionsv1beta1.RecreateDeploymentStrategyType,
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
			Spec: pgPodSpec(dbcr),
		},
	}
}

func pgPodSpec(dbcr *kciv1alpha1.Database) v1.PodSpec {
	return v1.PodSpec{
		Volumes: pgVolumes(dbcr),
		Containers: []v1.Container{
			pgContainerExporter(dbcr),
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

func pgContainerExporter(dbcr *kciv1alpha1.Database) v1.Container {
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

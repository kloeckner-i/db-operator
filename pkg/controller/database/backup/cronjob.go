package backup

import (
	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/config"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"

	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var conf = config.Config{}

// GCSBackupCron builds kubernetes cronjob object
// to create database backup regularly with defined schedule from dbcr
// this job will database dump and upload to google bucket storage for backup
func GCSBackupCron(dbcr *kciv1alpha1.Database) (*batchv1beta1.CronJob, error) {
	cronJobSpec, err := buildCronJobSpec(dbcr)
	if err != nil {
		return nil, err
	}

	return &batchv1beta1.CronJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CronJob",
			APIVersion: "batch",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbcr.Namespace + "-" + dbcr.Name + "-" + "backup",
			Namespace: dbcr.Namespace,
			Labels:    kci.BaseLabelBuilder(),
		},
		Spec: cronJobSpec,
	}, nil
}

func buildCronJobSpec(dbcr *kciv1alpha1.Database) (batchv1beta1.CronJobSpec, error) {
	jobTemplate, err := buildJobTemplate(dbcr)
	if err != nil {
		return batchv1beta1.CronJobSpec{}, err
	}

	return batchv1beta1.CronJobSpec{
		JobTemplate: jobTemplate,
		Schedule:    dbcr.Spec.Backup.Cron,
	}, nil
}

func buildJobTemplate(dbcr *kciv1alpha1.Database) (batchv1beta1.JobTemplateSpec, error) {
	ActiveDeadlineSeconds := int64(60 * 10) // 10m
	BackoffLimit := int32(3)
	backupContainer, err := buildBackupContainer(dbcr)
	if err != nil {
		return batchv1beta1.JobTemplateSpec{}, err
	}

	return batchv1beta1.JobTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: kci.BaseLabelBuilder(),
		},
		Spec: batchv1.JobSpec{
			ActiveDeadlineSeconds: &ActiveDeadlineSeconds,
			BackoffLimit:          &BackoffLimit,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: kci.BaseLabelBuilder(),
				},
				Spec: v1.PodSpec{
					Containers:    []v1.Container{backupContainer},
					NodeSelector:  conf.Backup.NodeSelector,
					RestartPolicy: v1.RestartPolicyNever,
					Volumes:       volumes(dbcr),
				},
			},
		},
	}, nil
}

func buildBackupContainer(dbcr *kciv1alpha1.Database) (v1.Container, error) {
	env, err := buildEnvVars(dbcr)
	if err != nil {
		return v1.Container{}, err
	}

	return v1.Container{
		Name:            "postgres-dump",
		Image:           conf.Backup.Postgres.Image,
		ImagePullPolicy: v1.PullAlways,
		VolumeMounts:    volumeMounts(),
		Env:             env,
	}, nil
}

func volumeMounts() []v1.VolumeMount {
	return []v1.VolumeMount{
		v1.VolumeMount{
			Name:      "gcloud-secret",
			MountPath: "/srv/gcloud/",
		},
		v1.VolumeMount{
			Name:      "postgres-cred",
			MountPath: "/srv/k8s/postgres-cred/",
		},
	}
}

func volumes(dbcr *kciv1alpha1.Database) []v1.Volume {
	return []v1.Volume{
		v1.Volume{
			Name: "gcloud-secret",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: "google-cloud-storage-bucket-cred",
				},
			},
		},
		v1.Volume{
			Name: "postgres-cred",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: dbcr.Spec.SecretName,
				},
			},
		},
	}
}

func buildEnvVars(dbcr *kciv1alpha1.Database) ([]v1.EnvVar, error) {
	instance, err := dbcr.GetInstanceRef()
	if err != nil {
		logrus.Errorf("can not build backup environment variables - %s", err)
		return nil, err
	}

	host := "localhost"
	if backend, _ := dbcr.GetBackendType(); backend == "google" {
		host = "db-" + dbcr.Name + "-svc"
	}

	return []v1.EnvVar{
		v1.EnvVar{
			Name: "DB_HOST", Value: host,
		},
		v1.EnvVar{
			Name: "DB_NAME", Value: dbcr.Namespace + "-" + dbcr.Name,
		},
		v1.EnvVar{
			Name: "DB_PASSWORD_FILE", Value: "/srv/k8s/postgres-cred/POSTGRES_PASSWORD",
		},
		v1.EnvVar{
			Name: "DB_USERNAME_FILE", Value: "/srv/k8s/postgres-cred/POSTGRES_USER",
		},
		v1.EnvVar{
			Name: "GCS_BUCKET", Value: instance.Spec.Backup.Bucket,
		},
	}, nil
}

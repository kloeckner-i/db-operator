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

package backup

import (
	"errors"
	"fmt"

	kciv1beta1 "github.com/db-operator/db-operator/api/v1beta1"
	"github.com/db-operator/db-operator/pkg/config"
	"github.com/db-operator/db-operator/pkg/utils/kci"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GCSBackupCron builds kubernetes cronjob object
// to create database backup regularly with defined schedule from dbcr
// this job will database dump and upload to google bucket storage for backup
func GCSBackupCron(conf *config.Config, dbcr *kciv1beta1.Database, ownership []metav1.OwnerReference) (*batchv1.CronJob, error) {
	cronJobSpec, err := buildCronJobSpec(conf, dbcr)
	if err != nil {
		return nil, err
	}

	return &batchv1.CronJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CronJob",
			APIVersion: "batch",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            dbcr.Namespace + "-" + dbcr.Name + "-" + "backup",
			Namespace:       dbcr.Namespace,
			Labels:          kci.BaseLabelBuilder(),
			OwnerReferences: ownership,
		},
		Spec: cronJobSpec,
	}, nil
}

func buildCronJobSpec(conf *config.Config, dbcr *kciv1beta1.Database) (batchv1.CronJobSpec, error) {
	jobTemplate, err := buildJobTemplate(conf, dbcr)
	if err != nil {
		return batchv1.CronJobSpec{}, err
	}

	return batchv1.CronJobSpec{
		JobTemplate: jobTemplate,
		Schedule:    dbcr.Spec.Backup.Cron,
	}, nil
}

func buildJobTemplate(conf *config.Config, dbcr *kciv1beta1.Database) (batchv1.JobTemplateSpec, error) {
	ActiveDeadlineSeconds := int64(conf.Backup.ActiveDeadlineSeconds)
	BackoffLimit := int32(3)
	instance, err := dbcr.GetInstanceRef()
	if err != nil {
		logrus.Errorf("can not build job template - %s", err)
		return batchv1.JobTemplateSpec{}, err
	}

	var backupContainer v1.Container

	engine := instance.Spec.Engine
	switch engine {
	case "postgres":
		backupContainer, err = postgresBackupContainer(conf, dbcr)
		if err != nil {
			return batchv1.JobTemplateSpec{}, err
		}
	case "mysql":
		backupContainer, err = mysqlBackupContainer(conf, dbcr)
		if err != nil {
			return batchv1.JobTemplateSpec{}, err
		}
	default:
		return batchv1.JobTemplateSpec{}, errors.New("unknown engine type")
	}

	return batchv1.JobTemplateSpec{
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

func getResourceRequirements(conf *config.Config) v1.ResourceRequirements {
	resourceRequirements := v1.ResourceRequirements{}

	requests := make(v1.ResourceList)
	cpuRequests, err := resource.ParseQuantity(conf.Backup.Resource.Requests.Cpu)
	if err == nil {
		requests["cpu"] = cpuRequests
	}
	memRequests, err := resource.ParseQuantity(conf.Backup.Resource.Requests.Memory)
	if err == nil {
		requests["memory"] = memRequests
	}
	if len(requests) != 0 {
		resourceRequirements.Requests = requests
	}

	limits := make(v1.ResourceList)
	cpuLimits, err := resource.ParseQuantity(conf.Backup.Resource.Limits.Cpu)
	if err == nil {
		limits["cpu"] = cpuLimits
	}
	memLimits, err := resource.ParseQuantity(conf.Backup.Resource.Limits.Memory)
	if err == nil {
		limits["memory"] = memLimits
	}
	if len(limits) != 0 {
		resourceRequirements.Limits = limits
	}

	return resourceRequirements
}

func postgresBackupContainer(conf *config.Config, dbcr *kciv1beta1.Database) (v1.Container, error) {
	env, err := postgresEnvVars(conf, dbcr)
	if err != nil {
		return v1.Container{}, err
	}

	return v1.Container{
		Name:            "postgres-dump",
		Image:           conf.Backup.Postgres.Image,
		ImagePullPolicy: v1.PullAlways,
		VolumeMounts:    volumeMounts(),
		Env:             env,
		Resources:       getResourceRequirements(conf),
	}, nil
}

func mysqlBackupContainer(conf *config.Config, dbcr *kciv1beta1.Database) (v1.Container, error) {
	env, err := mysqlEnvVars(dbcr)
	if err != nil {
		return v1.Container{}, err
	}

	return v1.Container{
		Name:            "mysql-dump",
		Image:           conf.Backup.Mysql.Image,
		ImagePullPolicy: v1.PullAlways,
		VolumeMounts:    volumeMounts(),
		Env:             env,
		Resources:       getResourceRequirements(conf),
	}, nil
}

func volumeMounts() []v1.VolumeMount {
	return []v1.VolumeMount{
		{
			Name:      "gcloud-secret",
			MountPath: "/srv/gcloud/",
		},
		{
			Name:      "db-cred",
			MountPath: "/srv/k8s/db-cred/",
		},
	}
}

func volumes(dbcr *kciv1beta1.Database) []v1.Volume {
	return []v1.Volume{
		{
			Name: "gcloud-secret",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: "google-cloud-storage-bucket-cred",
				},
			},
		},
		{
			Name: "db-cred",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: dbcr.Spec.SecretName,
				},
			},
		},
	}
}

func postgresEnvVars(conf *config.Config, dbcr *kciv1beta1.Database) ([]v1.EnvVar, error) {
	instance, err := dbcr.GetInstanceRef()
	if err != nil {
		logrus.Errorf("can not build backup environment variables - %s", err)
		return nil, err
	}

	host, err := getBackupHost(dbcr)
	if err != nil {
		return []v1.EnvVar{}, fmt.Errorf("can not build postgres backup job environment variables - %s", err)
	}

	port := instance.Status.Info["DB_PORT"]

	envList := []v1.EnvVar{
		{
			Name: "DB_HOST", Value: host,
		},
		{
			Name: "DB_PORT", Value: port,
		},
		{
			Name: "DB_NAME", ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: dbcr.Spec.SecretName},
					Key:                  "POSTGRES_DB",
				},
			},
		},
		{
			Name: "DB_PASSWORD_FILE", Value: "/srv/k8s/db-cred/POSTGRES_PASSWORD",
		},
		{
			Name: "DB_USERNAME_FILE", Value: "/srv/k8s/db-cred/POSTGRES_USER",
		},
		{
			Name: "GCS_BUCKET", Value: instance.Spec.Backup.Bucket,
		},
	}

	if instance.IsMonitoringEnabled() {
		envList = append(envList, v1.EnvVar{
			Name: "PROMETHEUS_PUSH_GATEWAY", Value: conf.Monitoring.PromPushGateway,
		})
	}

	return envList, nil
}

func mysqlEnvVars(dbcr *kciv1beta1.Database) ([]v1.EnvVar, error) {
	instance, err := dbcr.GetInstanceRef()
	if err != nil {
		logrus.Errorf("can not build backup environment variables - %s", err)
		return nil, err
	}

	host, err := getBackupHost(dbcr)
	if err != nil {
		return []v1.EnvVar{}, fmt.Errorf("can not build mysql backup job environment variables - %s", err)
	}
	port := instance.Status.Info["DB_PORT"]

	return []v1.EnvVar{
		{
			Name: "DB_HOST", Value: host,
		},
		{
			Name: "DB_PORT", Value: port,
		},
		{
			Name: "DB_NAME", ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: dbcr.Spec.SecretName},
					Key:                  "DB",
				},
			},
		},
		{
			Name: "DB_USER", ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: dbcr.Spec.SecretName},
					Key:                  "USER",
				},
			},
		},
		{
			Name: "DB_PASSWORD_FILE", Value: "/srv/k8s/db-cred/PASSWORD",
		},
		{
			Name: "GCS_BUCKET", Value: instance.Spec.Backup.Bucket,
		},
	}, nil
}

func getBackupHost(dbcr *kciv1beta1.Database) (string, error) {
	host := ""

	instance, err := dbcr.GetInstanceRef()
	if err != nil {
		return host, err
	}

	backend, err := dbcr.GetBackendType()
	if err != nil {
		return host, err
	}

	switch backend {
	case "google":
		host = "db-" + dbcr.Name + "-svc" // cloud proxy service name
		return host, nil
	case "generic":
		if instance.Spec.Generic.BackupHost != "" {
			return instance.Spec.Generic.BackupHost, nil
		}
		return instance.Spec.Generic.Host, nil
	default:
		return host, errors.New("unknown backend type")
	}
}

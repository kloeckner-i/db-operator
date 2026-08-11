/*
 * Copyright 2021 kloeckner.i GmbH
 * Copyright 2026 DB-Operator Authors
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

	kindav1beta1 "github.com/db-operator/db-operator/v2/api/v1beta1"
	"github.com/db-operator/db-operator/v2/pkg/config"
	"github.com/db-operator/db-operator/v2/pkg/utils/kci"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BackupCronJobManifest builds kubernetes cronjob object
// to create database backup regularly with defined schedule from dbcr
// this job will database dump and upload to google bucket storage for backup
func BackupCronJobManifest(conf *config.Config, dbcr *kindav1beta1.Database, instance *kindav1beta1.DbInstance) (*batchv1.CronJob, error) {
	cronJobSpec, err := buildCronJobSpec(conf, dbcr, instance)
	if err != nil {
		return nil, err
	}

	cronJob := &batchv1.CronJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CronJob",
			APIVersion: "batch",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbcr.Name + "-" + "backup",
			Namespace: dbcr.Namespace,
			Labels:    kci.BaseLabelBuilder(),
		},
		Spec: cronJobSpec,
	}

	return cronJob, nil
}

func buildCronJobSpec(conf *config.Config, dbcr *kindav1beta1.Database, instance *kindav1beta1.DbInstance) (batchv1.CronJobSpec, error) {
	jobTemplate, err := buildJobTemplate(conf, dbcr, instance)
	if err != nil {
		return batchv1.CronJobSpec{}, err
	}

	spec := batchv1.CronJobSpec{
		JobTemplate: jobTemplate,
		Schedule:    dbcr.Spec.Backup.Cron,
	}

	return spec, nil
}

func buildJobTemplate(conf *config.Config, dbcr *kindav1beta1.Database, instance *kindav1beta1.DbInstance) (batchv1.JobTemplateSpec, error) {
	var activeDeadlineSeconds int64
	if conf.Backup.ActiveDeadlineSeconds > 0 {
		activeDeadlineSeconds = int64(conf.Backup.ActiveDeadlineSeconds)
	} else {
		activeDeadlineSeconds = 600
	}
	backoffLimit := int32(3)

	var backupContainer v1.Container
	var err error

	engine := instance.Spec.Engine
	switch engine {
	case "postgres":
		backupContainer, err = postgresBackupContainer(conf, dbcr, instance)
		if err != nil {
			return batchv1.JobTemplateSpec{}, err
		}
	case "mysql":
		backupContainer, err = mysqlBackupContainer(conf, dbcr, instance)
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
			ActiveDeadlineSeconds: &activeDeadlineSeconds,
			BackoffLimit:          &backoffLimit,
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

func postgresBackupContainer(conf *config.Config, dbcr *kindav1beta1.Database, instance *kindav1beta1.DbInstance) (v1.Container, error) {
	env, err := postgresEnvVars(conf, dbcr, instance)
	if err != nil {
		return v1.Container{}, err
	}

	return v1.Container{
		Name:            "postgres-dump",
		Image:           conf.Backup.Postgres.Image,
		ImagePullPolicy: v1.PullAlways,
		VolumeMounts:    volumeMounts(),
		Env:             env,
		EnvFrom:         envFrom(dbcr),
		Resources:       *conf.Backup.Resources.DeepCopy(),
	}, nil
}

func mysqlBackupContainer(conf *config.Config, dbcr *kindav1beta1.Database, instance *kindav1beta1.DbInstance) (v1.Container, error) {
	env, err := mysqlEnvVars(dbcr, instance)
	if err != nil {
		return v1.Container{}, err
	}

	return v1.Container{
		Name:            "mysql-dump",
		Image:           conf.Backup.Mysql.Image,
		ImagePullPolicy: v1.PullAlways,
		VolumeMounts:    volumeMounts(),
		Env:             env,
		EnvFrom:         envFrom(dbcr),
		Resources:       *conf.Backup.Resources.DeepCopy(),
	}, nil
}

func volumeMounts() []v1.VolumeMount {
	mounts := []v1.VolumeMount{}

	dbCreds := &v1.VolumeMount{
		Name:      "db-cred",
		MountPath: "/srv/k8s/db-cred/",
	}
	mounts = append(mounts, *dbCreds)

	storage := &v1.VolumeMount{
		Name:      "storage",
		MountPath: "/backup",
	}
	mounts = append(mounts, *storage)
	return mounts
}

func volumes(dbcr *kindav1beta1.Database) []v1.Volume {
	volumes := []v1.Volume{}
	dbCreds := &v1.Volume{
		Name: "db-cred",
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: dbcr.Spec.SecretName,
			},
		},
	}
	volumes = append(volumes, *dbCreds)
	// TODO: Add PVC support
	storage := &v1.Volume{
		Name: "storage",
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{},
		},
	}
	volumes = append(volumes, *storage)
	return volumes
}

func postgresEnvVars(conf *config.Config, dbcr *kindav1beta1.Database, instance *kindav1beta1.DbInstance) ([]v1.EnvVar, error) {
	host, err := getBackupHost(dbcr, instance)
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
	}

	if instance.IsMonitoringEnabled() {
		envList = append(envList, v1.EnvVar{
			Name: "PROMETHEUS_PUSH_GATEWAY", Value: conf.Monitoring.PromPushGateway,
		})
	}

	if instance.Spec.Backup.Bucket != "" {
		envList = append(envList, v1.EnvVar{Name: "STORAGE_BUCKET", Value: instance.Spec.Backup.Bucket})
	}

	return envList, nil
}

func mysqlEnvVars(dbcr *kindav1beta1.Database, instance *kindav1beta1.DbInstance) ([]v1.EnvVar, error) {
	host, err := getBackupHost(dbcr, instance)
	if err != nil {
		return []v1.EnvVar{}, fmt.Errorf("can not build mysql backup job environment variables - %s", err)
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
	}

	if instance.Spec.Backup.Bucket != "" {
		envList = append(envList, v1.EnvVar{Name: "STORAGE_BUCKET", Value: instance.Spec.Backup.Bucket})
	} else {
		return nil, errors.New("backup bucket is not defined")
	}
	return envList, nil
}

func getBackupHost(dbcr *kindav1beta1.Database, instance *kindav1beta1.DbInstance) (string, error) {
	host := ""

	backend, err := instance.GetBackendType()
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

// This function is used to set env variables in the backup pod.
// Currently it's only possible to set a secret as a source.
// By default it's supposed to be used by rclone to be able to
// upload backup to the storage
func envFrom(dbcr *kindav1beta1.Database) []v1.EnvFromSource {
	envFrom := []v1.EnvFromSource{}
	optional := false
	if dbcr.Spec.Backup.EnvFromSecret != "" {
		envFromSecret := v1.EnvFromSource{
			SecretRef: &v1.SecretEnvSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: dbcr.Spec.Backup.EnvFromSecret,
				},
				Optional: &optional,
			},
		}
		envFrom = append(envFrom, envFromSecret)
	}
	return envFrom
}

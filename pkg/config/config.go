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

package config

import (
	"os"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

// Config defines configurations needed by db-operator
type Config struct {
	Instances  instanceConfig   `yaml:"instance"`
	Backup     *BackupConfig    `yaml:"backup"`
	Monitoring monitoringConfig `yaml:"monitoring"`
}
type instanceConfig struct {
	Google  googleInstanceConfig  `yaml:"google"`
	Generic genericInstanceConfig `yaml:"generic"`
	Percona perconaClusterConfig  `yaml:"percona"`
}

type googleInstanceConfig struct {
	ClientSecretName string      `yaml:"clientSecretName"`
	ProxyConfig      proxyConfig `yaml:"proxy"`
}

type genericInstanceConfig struct { // TODO
}

type perconaClusterConfig struct {
	ProxyConfig proxyConfig `yaml:"proxy"`
}

type proxyConfig struct {
	NodeSelector map[string]string `yaml:"nodeSelector"`
	Image        string            `yaml:"image"`
	MetricsPort  int               `yaml:"metricsPort"`
}

// BackupConfig defines docker image for creating database dump by backup cronjob
// backup cronjob will be created by db-operator when backup is enabled
type BackupConfig struct {
	Postgres              *postgresBackupConfig        `yaml:"postgres"`
	Mysql                 *mysqlBackupConfig           `yaml:"mysql"`
	NodeSelector          map[string]string            `yaml:"nodeSelector"`
	ActiveDeadlineSeconds int64                        `default:"1200" yaml:"activeDeadlineSeconds"`
	Resources             *corev1.ResourceRequirements `yaml:"resources"`
	// Must be in the same namespace as the DB Operator
	StorageCredSecret string `yaml:"storageCredSecret"`
}

type postgresBackupConfig struct {
	Image string `yaml:"image"`
}

type mysqlBackupConfig struct {
	Image string `yaml:"image"`
}

// monitoringConfig defines prometheus exporter configurations
// which will be created by db-operator when monitoring is enabled
type monitoringConfig struct {
	PromPushGateway string `yaml:"promPushGateway,omitempty"`
}

// LoadConfig reads config file for db-operator from defined path and parse
func LoadConfig(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	conf := &Config{}

	if err = yaml.Unmarshal(data, &conf); err != nil {
		return nil, err
	}
	return conf, nil
}

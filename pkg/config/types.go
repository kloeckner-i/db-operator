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

// Config defines configurations needed by db-operator
type Config struct {
	Instances  instanceConfig   `yaml:"instance"`
	Backup     backupConfig     `yaml:"backup"`
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
}

// backupConfig defines docker image for creating database dump by backup cronjob
// backup cronjob will be created by db-operator when backup is enabled
type backupConfig struct {
	Postgres              postgresBackupConfig `yaml:"postgres"`
	Mysql                 mysqlBackupConfig    `yaml:"mysql"`
	NodeSelector          map[string]string    `yaml:"nodeSelector"`
	ActiveDeadlineSeconds int64                `yaml:"activeDeadlineSeconds"`
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
	Postgres        postgresMonitoringConfig `yaml:"postgres"`
	Mysql           mysqlMonitoringConfig    `yaml:"mysql,omitempty"`
	NodeSelector    map[string]string        `yaml:"nodeSelector"`
	PromPushGateway string                   `yaml:"promPushGateway,omitempty"`
}

type postgresMonitoringConfig struct {
	ExporterImage string `yaml:"image"`
	Queries       string `yaml:"queries"`
}

type mysqlMonitoringConfig struct { // TODO
}

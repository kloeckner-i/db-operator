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
}

type googleInstanceConfig struct {
	ClientSecretName string      `yaml:"clientSecretName"`
	ProxyConfig      proxyConfig `yaml:"proxy"`
}

type genericInstanceConfig struct {
	// TODO
}

type proxyConfig struct {
	NodeSelector map[string]string `yaml:"nodeSelector"`
	Image        string            `yaml:"image"`
}

// backupConfig defines docker image for creating database dump by backup cronjob
// backup cronjob will be created by db-operator when backup is enabled
type backupConfig struct {
	Postgres     postgresBackupConfig `yaml:"postgres"`
	Mysql        mysqlBackupConfig    `yaml:"mysql"`
	NodeSelector map[string]string    `yaml:"nodeSelector"`
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
	Postgres     postgresMonitoringConfig `yaml:"postgres"`
	Mysql        mysqlMonitoringConfig    `yaml:"mysql,omitempty"`
	NodeSelector map[string]string        `yaml:"nodeSelector"`
}

type postgresMonitoringConfig struct {
	ExporterImage string `yaml:"image"`
	Queries       string `yaml:"queries"`
}

type mysqlMonitoringConfig struct {
	// TODO
}

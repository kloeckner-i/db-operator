package config

import (
	"io/ioutil"
	"os"

	"github.com/kloeckner-i/db-operator/pkg/utils/kci"

	"github.com/sirupsen/logrus"

	yaml "gopkg.in/yaml.v2"
)

// LoadConfig reads config file for db-operator from defined path and parse
func LoadConfig() Config {
	path := kci.StringNotEmpty(os.Getenv("CONFIG_PATH"), "/srv/config/config.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		logrus.Fatalf("Failed to open config file: %v", err)
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		logrus.Fatalf("Loading of configuration failed: %v", err)
	}

	conf := Config{}

	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		logrus.Fatalf("Decode of configuration failed: %v", err)
	}
	return conf
}

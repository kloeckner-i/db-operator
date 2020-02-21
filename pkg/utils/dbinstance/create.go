package dbinstance

import (
	"github.com/sirupsen/logrus"
)

// Create instance if not exists
func Create(ins DbInstance) (map[string]string, error) {
	err := ins.exist()
	if err == nil {
		return nil, ErrAlreadyExists
	}

	logrus.Debug("instance doesn't exist, create instance")
	err = ins.create()
	if err != nil {
		logrus.Debug("creation failed")
		return nil, err
	}

	data, err := ins.getInfoMap()
	if err != nil {
		return nil, err
	}

	return data, nil
}

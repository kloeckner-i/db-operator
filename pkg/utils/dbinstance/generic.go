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

package dbinstance

import (
	"errors"
	"strconv"

	kcidb "github.com/kloeckner-i/db-operator/pkg/utils/database"

	"github.com/sirupsen/logrus"
)

// Generic represents database instance which can be connected by address and port
type Generic struct {
	Host         string
	Port         uint16
	Engine       string
	User         string
	Password     string
	PublicIP     string
	SSLEnabled   bool
	SkipCAVerify bool
}

func makeInterface(in *Generic) (kcidb.Database, error) {
	switch in.Engine {
	case "postgres":
		db := kcidb.Postgres{
			Host:         in.Host,
			Port:         in.Port,
			User:         in.User,
			Password:     in.Password,
			Database:     "postgres",
			SSLEnabled:   in.SSLEnabled,
			SkipCAVerify: in.SkipCAVerify,
		}
		return db, nil
	case "mysql":
		db := kcidb.Mysql{
			Host:         in.Host,
			Port:         in.Port,
			User:         in.User,
			Password:     in.Password,
			Database:     "mysql",
			SSLEnabled:   in.SSLEnabled,
			SkipCAVerify: in.SkipCAVerify,
		}
		return db, nil
	default:
		return nil, errors.New("not supported engine type")
	}
}

func (ins *Generic) state() (string, error) {
	logrus.Debug("generic db instance not support a state check")
	return "", nil
}

func (ins *Generic) exist() error {
	db, err := makeInterface(ins)
	if err != nil {
		logrus.Errorf("can not check if instance exists because of %s", err)
		return err
	}
	err = db.CheckStatus()
	if err != nil {
		logrus.Error(err)
		return err
	}
	return nil // instance exist
}

func (ins *Generic) create() error {
	return errors.New("creating generic db instance is not yet implimented")
}

func (ins *Generic) update() error {
	logrus.Debug("updating generic db instance is not yet implimented")
	return nil
}

func (ins *Generic) getInfoMap() (map[string]string, error) {
	data := map[string]string{
		"DB_CONN":      ins.Host,
		"DB_PORT":      strconv.FormatInt(int64(ins.Port), 10),
		"DB_PUBLIC_IP": ins.PublicIP,
	}

	return data, nil
}

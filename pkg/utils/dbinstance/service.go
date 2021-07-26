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

// Update instance if instance exists
func Update(ins DbInstance) (map[string]string, error) {
	err := ins.exist()
	if err != nil {
		return nil, ErrNotExists
	}

	state, err := ins.state()
	if err != nil {
		return nil, err
	}

	if state != "RUNNABLE" {
		return nil, ErrInstanceNotReady
	}

	err = ins.update()
	if err != nil {
		logrus.Debug("update failed")
		return nil, err
	}

	data, err := ins.getInfoMap()
	if err != nil {
		return nil, err
	}

	return data, nil
}

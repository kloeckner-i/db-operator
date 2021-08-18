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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kloeckner-i/db-operator/pkg/utils/gcloud"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
)

// Gsql represents a google sql instance
type Gsql struct {
	Name        string
	Config      string
	User        string
	Password    string
	ProjectID   string
	APIEndpoint string
}

// GsqlNew create a new Gsql object and return
func GsqlNew(name, config, user, password, apiEndpoint string) *Gsql {
	projectID := gcloud.GetServiceAccount().ProjectID

	return &Gsql{
		Name:        name,
		Config:      config,
		User:        user,
		Password:    password,
		ProjectID:   projectID,
		APIEndpoint: apiEndpoint,
	}
}

func (ins *Gsql) getSqladminService(ctx context.Context) (*sqladmin.Service, error) {
	opts := []option.ClientOption{}

	// if APIEndpoint is defined, it considered as test mode and disable oauth
	if ins.APIEndpoint != "" {
		opts = append(opts, option.WithEndpoint(ins.APIEndpoint))
		opts = append(opts, option.WithHTTPClient(oauth2.NewClient(ctx, &disabledTokenSource{})))
	}

	sqladminService, err := sqladmin.NewService(ctx, opts...)
	if err != nil {
		logrus.Debugf("error occurs during getting sqladminService %s", err)
		return nil, err
	}

	return sqladminService, nil
}

func (ins *Gsql) getInstance() (*sqladmin.DatabaseInstance, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sqladminService, err := ins.getSqladminService(ctx)
	if err != nil {
		return nil, err
	}

	rs, err := sqladminService.Instances.Get(ins.ProjectID, ins.Name).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	return rs, nil
}

func (ins *Gsql) createInstance() error {
	logrus.Debugf("gsql instance create %s", ins.Name)
	request, err := ins.verifyConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sqladminService, err := ins.getSqladminService(ctx)
	if err != nil {
		return err
	}

	// Project ID of the project to which the newly created Cloud SQL instances should belong.
	resp, err := sqladminService.Instances.Insert(ins.ProjectID, request).Context(ctx).Do()
	if err != nil {
		logrus.Errorf("gsql instance insert error - %s", err)
		return err
	}
	logrus.Debugf("instance insert api response: %#v", resp)
	err = ins.waitUntilRunnable()
	if err != nil {
		return fmt.Errorf("gsql instance created but still not runnable - %s", err)
	}

	return err
}

func (ins *Gsql) updateInstance() error {
	logrus.Debugf("gsql instance create %s", ins.Name)
	request, err := ins.verifyConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sqladminService, err := ins.getSqladminService(ctx)
	if err != nil {
		return err
	}

	// Project ID of the project to which the newly created Cloud SQL instances should belong.
	resp, err := sqladminService.Instances.Patch(ins.ProjectID, ins.Name, request).Context(ctx).Do()
	if err != nil {
		logrus.Errorf("gsql instance patch error - %s", err)
		return err
	}
	logrus.Debugf("instance patch api response: %#v", resp)

	err = ins.waitUntilRunnable()
	if err != nil {
		return fmt.Errorf("gsql instance created but still not runnable - %s", err)
	}

	return err
}

func (ins *Gsql) updateUser() error {
	logrus.Debugf("gsql user update - instance: %s, user: %s", ins.Name, ins.User)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sqladminService, err := ins.getSqladminService(ctx)
	if err != nil {
		return err
	}

	host := "%"
	rb := &sqladmin.User{
		Host:     host,
		Name:     ins.User,
		Password: ins.Password,
	}

	resp, err := sqladminService.Users.Update(ins.ProjectID, ins.Name, rb).Host(host).Name(ins.User).Context(ctx).Do()
	if err != nil {
		return err
	}
	logrus.Debugf("user update api response: %#v", resp)

	err = ins.waitUntilRunnable()
	if err != nil {
		return fmt.Errorf("gsql user updated but still not runnable - %s", err)
	}

	return nil
}

func (ins *Gsql) verifyConfig() (*sqladmin.DatabaseInstance, error) {
	// require non empty name and config
	rb := &sqladmin.DatabaseInstance{}
	err := json.Unmarshal([]byte(ins.Config), rb)
	if err != nil {
		logrus.Errorf("can not verify config - %s", err)
		logrus.Debugf("%#v\n", []byte(ins.Config))
		return nil, err
	}
	rb.Name = ins.Name
	return rb, nil
}

func (ins *Gsql) waitUntilRunnable() error {
	const delay = 30
	time.Sleep(delay * time.Second)

	err := kci.Retry(10, 60*time.Second, func() error {
		state, err := ins.state()
		if err != nil {
			return err
		}
		if state != "RUNNABLE" {
			return errors.New("gsql instance not ready yet")
		}

		return nil
	})
	if err != nil {
		instance, err := ins.getInstance()
		if err != nil {
			return err
		}

		return fmt.Errorf("gsql instance state not ready %s", instance.State)
	}

	return nil
}

func (ins *Gsql) state() (string, error) {
	instance, err := ins.getInstance()
	if err != nil {
		return "", err
	}
	logrus.Debugf("check gsql instance %s state: %s", ins.Name, instance.State)
	return instance.State, nil
}

func (ins *Gsql) create() error {
	err := ins.createInstance()
	if err != nil {
		logrus.Errorf("gsql instance creation error - %s", err)
		return err
	}

	err = ins.updateUser()
	if err != nil {
		logrus.Errorf("gsql user update error - %s", err)
		return err
	}

	return nil
}

func (ins *Gsql) update() error {
	err := ins.updateInstance()
	if err != nil {
		logrus.Errorf("gsql instance update error - %s", err)
		return err
	}

	err = ins.updateUser()
	if err != nil {
		logrus.Errorf("gsql user update error - %s", err)
		return err
	}

	return nil
}

func (ins *Gsql) exist() error {
	_, err := ins.getInstance()
	if err != nil {
		logrus.Debugf("gsql instance get failed %s", err)
		return err
	}
	return nil // instance exist
}

func (ins *Gsql) getInfoMap() (map[string]string, error) {
	instance, err := ins.getInstance()
	if err != nil {
		return nil, err
	}

	data := map[string]string{
		"DB_INSTANCE":  instance.Name,
		"DB_CONN":      instance.ConnectionName,
		"DB_PUBLIC_IP": getGsqlPublicIP(instance),
		"DB_PORT":      determineGsqlPort(instance),
		"DB_VERSION":   instance.DatabaseVersion,
	}

	return data, nil
}

func getGsqlPublicIP(instance *sqladmin.DatabaseInstance) string {
	for _, ip := range instance.IpAddresses {
		if ip.Type == "PRIMARY" {
			return ip.IpAddress
		}
	}

	return "-"
}

func determineGsqlPort(instance *sqladmin.DatabaseInstance) string {
	databaseVersion := strings.ToLower(instance.DatabaseVersion)
	if strings.Contains(databaseVersion, "postgres") {
		return "5432"
	}

	if strings.Contains(databaseVersion, "mysql") {
		return "3306"
	}

	return "-"
}

// disabledTokenSource is a mocked oauth token source for local testing.
type disabledTokenSource struct{}

// Token issues a mocked bearer token for local testing.
func (ts *disabledTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		TokenType: "Bearer",
		Expiry:    time.Now().Add(time.Hour),
	}, nil
}

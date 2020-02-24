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
	"golang.org/x/oauth2/google"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
)

// Gsql represents a google sql instance
type Gsql struct {
	Name     string
	Config   string
	User     string
	Password string
}

func getSqladminService(ctx context.Context) (*sqladmin.Service, error) {
	oauthClient, err := google.DefaultClient(ctx, sqladmin.CloudPlatformScope)
	if err != nil {
		logrus.Errorf("failed to get google auth client %s", err)
		return nil, err
	}

	sqladminService, err := sqladmin.New(oauthClient)
	if err != nil {
		logrus.Debugf("error occurs during getting sqladminService %s", err)
		return nil, err
	}

	return sqladminService, nil
}

func getGsqlInstance(name string) (*sqladmin.DatabaseInstance, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sqladminService, err := getSqladminService(ctx)
	if err != nil {
		return nil, err
	}

	serviceaccount := gcloud.GetServiceAccount()

	rs, err := sqladminService.Instances.Get(serviceaccount.ProjectID, name).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	return rs, nil
}

func updateGsqlUser(instance, user, password string) error {
	logrus.Debugf("gsql user update - instance: %s, user: %s", instance, user)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sqladminService, err := getSqladminService(ctx)
	if err != nil {
		return err
	}

	host := "%"
	rb := &sqladmin.User{
		Password: password,
	}

	serviceaccount := gcloud.GetServiceAccount()
	project := serviceaccount.ProjectID
	resp, err := sqladminService.Users.Update(project, instance, user, rb).Host(host).Context(ctx).Do()
	if err != nil {
		return err
	}
	logrus.Debugf("user update api response: %#v", resp)

	return nil
}

func (ins *Gsql) verifyConfig() (*sqladmin.DatabaseInstance, error) {
	//require non empty name and config
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
		instance, err := getGsqlInstance(ins.Name)
		if err != nil {
			return err
		}
		logrus.Debugf("waiting gsql instance %s state: %s", ins.Name, instance.State)

		if instance.State != "RUNNABLE" {
			return errors.New("gsql instance not ready yet")
		}

		return nil
	})
	if err != nil {
		instance, err := getGsqlInstance(ins.Name)
		if err != nil {
			return err
		}

		return fmt.Errorf("gsql instance state not ready %s", instance.State)
	}

	return nil
}

func (ins *Gsql) create() error {
	logrus.Debugf("gsql instance create %s", ins.Name)
	request, err := ins.verifyConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sqladminService, err := getSqladminService(ctx)
	if err != nil {
		return err
	}

	// Project ID of the project to which the newly created Cloud SQL instances should belong.
	serviceaccount := gcloud.GetServiceAccount()
	project := serviceaccount.ProjectID

	resp, err := sqladminService.Instances.Insert(project, request).Context(ctx).Do()
	if err != nil {
		logrus.Errorf("gsql instance insert error - %s", err)
		return err
	}
	logrus.Debugf("instance insert api response: %#v", resp)

	err = ins.waitUntilRunnable()
	if err != nil {
		return fmt.Errorf("gsql instance created but still not runnable - %s", err)
	}

	err = updateGsqlUser(ins.Name, ins.User, ins.Password)
	if err != nil {
		logrus.Errorf("gsql user update error - %s", err)
		return err
	}

	return nil
}

func (ins *Gsql) update() error {
	logrus.Debugf("gsql instance update %s", ins.Name)
	request, err := ins.verifyConfig()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sqladminService, err := getSqladminService(ctx)
	if err != nil {
		return err
	}

	// Project ID of the project to which the newly created Cloud SQL instances should belong.
	serviceaccount := gcloud.GetServiceAccount()
	project := serviceaccount.ProjectID

	resp, err := sqladminService.Instances.Patch(project, ins.Name, request).Context(ctx).Do()
	if err != nil {
		logrus.Errorf("gsql instance patch error - %s", err)
		return err
	}
	logrus.Debugf("instance patch api response: %#v", resp)

	err = ins.waitUntilRunnable()
	if err != nil {
		return fmt.Errorf("gsql instance updated but still not runnable - %s", err)
	}

	err = updateGsqlUser(ins.Name, ins.User, ins.Password)
	if err != nil {
		logrus.Errorf("gsql user update error - %s", err)
		return err
	}

	return nil
}

func (ins *Gsql) exist() error {
	_, err := getGsqlInstance(ins.Name)
	if err != nil {
		logrus.Debugf("gsql instance get failed %s", err)
		return err
	}
	return nil // instance exist
}

func (ins *Gsql) getInfoMap() (map[string]string, error) {
	instance, err := getGsqlInstance(ins.Name)
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

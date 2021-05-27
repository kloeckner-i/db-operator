package dbinstance

import (
	"errors"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	database "github.com/kloeckner-i/db-operator/pkg/utils/database"
	"github.com/kloeckner-i/db-operator/pkg/utils/dbinstance"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	"github.com/sirupsen/logrus"
)

func (r *ReconcileDbInstance) create(dbin *kciv1alpha1.DbInstance) error {
	secret, err := kci.GetSecretResource(dbin.Spec.AdminUserSecret)
	if err != nil {
		logrus.Errorf("Instance: name=%s failed to get instance admin user secret %s/%s", dbin.Name, dbin.Spec.AdminUserSecret.Namespace, dbin.Spec.AdminUserSecret.Name)
		return err
	}

	db := database.New(dbin.Spec.Engine)
	cred, err := db.ParseAdminCredentials(secret.Data)
	if err != nil {
		return err
	}

	backend, err := dbin.GetBackendType()
	if err != nil {
		return err
	}

	var instance dbinstance.DbInstance
	switch backend {
	case "google":
		configmap, err := kci.GetConfigResource(dbin.Spec.Google.ConfigmapName)
		if err != nil {
			logrus.Errorf("Instance: name=%s reading GCSQL instance config %s/%s", dbin.Name, dbin.Spec.Google.ConfigmapName.Namespace, dbin.Spec.Google.ConfigmapName.Name)
			return err
		}

		name := dbin.Spec.Google.InstanceName
		config := configmap.Data["config"]
		user := cred.Username
		password := cred.Password

		instance = dbinstance.GsqlNew(name, config, user, password)
	case "generic":
		instance = &dbinstance.Generic{
			Host:         dbin.Spec.Generic.Host,
			Port:         dbin.Spec.Generic.Port,
			PublicIP:     dbin.Spec.Generic.PublicIP,
			Engine:       dbin.Spec.Engine,
			User:         cred.Username,
			Password:     cred.Password,
			SSLEnabled:   dbin.Spec.SSLConnection.Enabled,
			SkipCAVerify: dbin.Spec.SSLConnection.SkipVerify,
		}
	case "percona":
		if dbin.Spec.Engine != "mysql" {
			logrus.Errorf("Instance: name=%s - non mysql percona instance not supported", dbin.Name)
			return errors.New("non mysql percona instance not supported")
		}

		instance = &dbinstance.Generic{
			Host:         dbin.Spec.Percona.ServerList[0].Host,
			Port:         dbin.Spec.Percona.ServerList[0].Port,
			Engine:       dbin.Spec.Engine,
			User:         cred.Username,
			Password:     cred.Password,
			SSLEnabled:   dbin.Spec.SSLConnection.Enabled,
			SkipCAVerify: dbin.Spec.SSLConnection.SkipVerify,
		}
	default:
		return errors.New("not supported backend type")
	}

	info, err := dbinstance.Create(instance)
	if err != nil {
		if err == dbinstance.ErrAlreadyExists {
			logrus.Debugf("Instance: name=%s instance already exists in backend, updating instance", dbin.Name)
			info, err = dbinstance.Update(instance)
			if err != nil {
				logrus.Errorf("Instance: name=%s failed updating instance - %s", dbin.Name, err)
				return err
			}
		} else {
			logrus.Errorf("Instance: name=%s failed creating instance - %s", dbin.Name, err)
			return err
		}
	}

	dbin.Status.Info = info
	return nil
}

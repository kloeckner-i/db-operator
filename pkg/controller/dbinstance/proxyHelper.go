package dbinstance

import (
	"errors"
	"github.com/kloeckner-i/db-operator/pkg/config"
	"strconv"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/pkg/apis/kci/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	proxy "github.com/kloeckner-i/db-operator/pkg/utils/proxy"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"github.com/sirupsen/logrus"
)

var (
	// ErrNoProxySupport is thrown when proxy creation is not supported
	ErrNoProxySupport = errors.New("no proxy supported backend type")
)

func determinProxyType(conf *config.Config, dbin *kciv1alpha1.DbInstance) (proxy.Proxy, error) {
	logrus.Debugf("Instance: name=%s - determinProxyType", dbin.Name)
	operatorNamespace, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		// can not get operator namespace
		return nil, err
	}

	backend, err := dbin.GetBackendType()
	if err != nil {
		return nil, err
	}

	switch backend {
	case "google":
		portString := dbin.Status.Info["DB_PORT"]
		port, err := strconv.Atoi(portString)
		if err != nil {
			logrus.Errorf("can not convert DB_PORT to int - %s", err)
			return nil, err
		}

		labels := map[string]string{
			"app":           "cloudproxy",
			"instance-name": dbin.Name,
		}

		return &proxy.CloudProxy{
			NamePrefix:             "dbinstance-" + dbin.Name,
			Namespace:              operatorNamespace,
			InstanceConnectionName: dbin.Status.Info["DB_CONN"],
			AccessSecretName:       conf.Instances.Google.ClientSecretName,
			Engine:                 dbin.Spec.Engine,
			Port:                   int32(port),
			Labels:                 kci.LabelBuilder(labels),
		}, nil
	default:
		return nil, ErrNoProxySupport
	}
}

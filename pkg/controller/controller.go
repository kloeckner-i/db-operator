package controller

import (
	"github.com/kloeckner-i/db-operator/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(*config.Config, manager.Manager) error

// AddToManager adds all Controllers to the Manager
func AddToManager(conf *config.Config, m manager.Manager) error {
	for _, f := range AddToManagerFuncs {
		if err := f(conf, m); err != nil {
			return err
		}
	}
	return nil
}

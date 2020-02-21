package controller

import (
	"github.com/kloeckner-i/db-operator/pkg/controller/dbinstance"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, dbinstance.Add)
}

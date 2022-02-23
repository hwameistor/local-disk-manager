package controller

import (
	"github.com/hwameistor/local-disk-manager/pkg/controller/localdisk"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, localdisk.Add)
}

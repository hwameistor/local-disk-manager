package controller

import (
	"github.com/hwameistor/local-disk-manager/pkg/controller/localdiskclaim"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, localdiskclaim.Add)
}

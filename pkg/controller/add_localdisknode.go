package controller

import (
	"github.com/hwameistor/local-disk-manager/pkg/controller/localdisknode"
)

func init() {
	// AddToNodeManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToNodeManagerFuncs = append(AddToNodeManagerFuncs, localdisknode.Add)
}

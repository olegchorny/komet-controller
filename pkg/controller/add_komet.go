package controller

import (
	"github.com/plerionio/komet-controller/pkg/controller/komet"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, komet.Add)
}

package controller

import (
	"github.com/WianVos/selenium-operator/pkg/controller/seleniumgrid"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, seleniumgrid.Add)
}

package main

import (
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-role-controller/pkg/controller"
	"github.com/jenkins-x/jx-role-controller/pkg/loghelpers"
)

func main() {
	loghelpers.InitLogrus()

	roleController, err := controller.NewRoleController()
	if err != nil {
		log.Logger().Fatalf(err.Error())
	}
	err = roleController.Run()
	if err != nil {
		log.Logger().Fatalf(err.Error())
	}
}

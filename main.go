package main

import (
	"github.com/jenkins-x/jx-role-controller/pkg/controller"
	"github.com/jenkins-x/jx-role-controller/pkg/loghelpers"
	"github.com/sirupsen/logrus"
)

func main() {
	loghelpers.InitLogrus()

	roleController, err := controller.NewRoleController()
	if err != nil {
		logrus.Fatalf(err.Error())
	}
	err = roleController.Run()
	if err != nil {
		logrus.Fatalf(err.Error())
	}
}

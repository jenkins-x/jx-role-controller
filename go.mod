module github.com/jenkins-x/jx-role-controller

go 1.13

require (
	github.com/fatih/color v1.9.0
	github.com/jenkins-x/jx-api v0.0.6
	github.com/jenkins-x/jx-kube-client v0.0.7
	github.com/jenkins-x/jx-logging v0.0.8
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	k8s.io/api v0.18.4
	k8s.io/apimachinery v0.18.4
	k8s.io/client-go v11.0.0+incompatible
)

replace k8s.io/api => k8s.io/api v0.16.5

replace k8s.io/metrics => k8s.io/metrics v0.16.5

replace k8s.io/apimachinery => k8s.io/apimachinery v0.16.5

replace k8s.io/client-go => k8s.io/client-go v0.16.5

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20181128195303-1f84094d7e8e

replace github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v21.1.0+incompatible

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.1.1+incompatible

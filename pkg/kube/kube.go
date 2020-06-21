package kube

import (
	"github.com/jenkins-x/jx-kube-client/pkg/kubeclient"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func NewClientAndConfig() (kubernetes.Interface, *rest.Config, error) {
	factory := kubeclient.NewFactory()
	config, err := factory.CreateKubeConfig()
	if err != nil {
		log.Logger().Fatalf("failed to get kubernetes config: %v", err)
		return nil, nil, errors.WithStack(err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Logger().Fatalf("error building kubernetes clientset: %v", err)
		return nil, nil, errors.WithStack(err)
	}
	return client, config, nil
}

package kube

import (
	"strings"

	v1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewPermanentEnvironment creates a new permanent environment for testing
func NewPermanentEnvironment(name string) *v1.Environment {
	return &v1.Environment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "jx",
		},
		Spec: v1.EnvironmentSpec{
			Label:             strings.Title(name),
			Namespace:         "jx-" + name,
			PromotionStrategy: v1.PromotionStrategyTypeAutomatic,
			Kind:              v1.EnvironmentKindTypePermanent,
		},
	}
}

// NewPreviewEnvironment creates a new preview environment for testing
func NewPreviewEnvironment(name string) *v1.Environment {
	return &v1.Environment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "jx",
		},
		Spec: v1.EnvironmentSpec{
			Label:             strings.Title(name),
			Namespace:         "jx-preview-" + name,
			PromotionStrategy: v1.PromotionStrategyTypeAutomatic,
			Kind:              v1.EnvironmentKindTypePreview,
		},
	}
}

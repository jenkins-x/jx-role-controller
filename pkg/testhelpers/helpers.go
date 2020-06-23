package testhelpers

import (
	"os"
	"strings"

	v1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	v1fake "github.com/jenkins-x/jx-api/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx-role-controller/pkg/controller"
	"github.com/jenkins-x/jx-role-controller/pkg/kube"
	"github.com/jenkins-x/jx-role-controller/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// IsDebugLog debug log?
func IsDebugLog() bool {
	return strings.ToLower(os.Getenv("JX_TEST_DEBUG")) == "true"
}

// Debugf debug format
func Debugf(message string, args ...interface{}) {
	if IsDebugLog() {
		log.Logger().Infof(message, args...)
	}
}

// ConfigureTestOptions lets configure the options for use in tests
// using fake APIs to k8s cluster.
func ConfigureTestOptionsWithResources(o *controller.RoleOptions, k8sObjects []runtime.Object, jxObjects []runtime.Object) {
	currentNamespace := "jx"
	o.TeamNs = currentNamespace

	namespacesRequired := []string{currentNamespace}
	namespaceMap := map[string]*corev1.Namespace{}

	for _, ro := range k8sObjects {
		ns, ok := ro.(*corev1.Namespace)
		if ok {
			namespaceMap[ns.Name] = ns
		}
	}
	hasDev := false
	for _, ro := range jxObjects {
		env, ok := ro.(*v1.Environment)
		if ok {
			ns := env.Spec.Namespace
			if ns != "" && util.StringArrayIndex(namespacesRequired, ns) < 0 {
				namespacesRequired = append(namespacesRequired, ns)
			}
			if env.Name == "dev" {
				hasDev = true
			}
		}
	}

	// ensure we've the dev environment
	if !hasDev {
		devEnv := kube.NewPermanentEnvironment("dev")
		devEnv.Spec.Namespace = currentNamespace
		devEnv.Spec.Kind = v1.EnvironmentKindTypeDevelopment

		jxObjects = append(jxObjects, devEnv)
	}

	// add any missing namespaces
	for _, ns := range namespacesRequired {
		if namespaceMap[ns] == nil {
			k8sObjects = append(k8sObjects, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
					Labels: map[string]string{
						"tag": "",
					},
				},
			})
		}
	}

	client := fake.NewSimpleClientset(k8sObjects...)
	o.KubeClient = client
	o.JxClient = v1fake.NewSimpleClientset(jxObjects...)
}

package controller_test

import (
	"fmt"
	"testing"

	"github.com/jenkins-x/jx-role-controller/pkg/controller"
	"github.com/jenkins-x/jx-role-controller/pkg/kube"
	"github.com/jenkins-x/jx-role-controller/pkg/testhelpers"
	"github.com/jenkins-x/jx-role-controller/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"

	v1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-logging/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

func Test_EnvironmentRoleBinding(t *testing.T) {
	t.Parallel()
	o := &controller.RoleOptions{
		NoWatch: true,
	}
	roleName := "myrole"
	roleBindingName := roleName
	roleNameWithoutLabel := "myroleWithoutLabel"
	teamNs := "jx"
	roleLabels := make(map[string]string)
	roleLabels[kube.LabelKind] = kube.ValueKindEnvironmentRole

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: teamNs,
			Labels:    roleLabels,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "watch", "list"},
				APIGroups: []string{""},
				Resources: []string{"configmaps", "pods", "services"},
			},
		},
	}

	roleWithLabel := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleNameWithoutLabel,
			Namespace: teamNs,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "watch", "list"},
				APIGroups: []string{""},
				Resources: []string{"configmaps", "pods", "services"},
			},
		},
	}

	envRoleBinding := &v1.EnvironmentRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: teamNs,
		},
		Spec: v1.EnvironmentRoleBindingSpec{
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "jenkins",
					Namespace: teamNs,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     roleName,
			},
			Environments: []v1.EnvironmentFilter{
				{
					Includes: []string{"*"},
				},
			},
		},
	}

	testhelpers.ConfigureTestOptionsWithResources(o,
		[]runtime.Object{
			role,
			roleWithLabel,
		},
		[]runtime.Object{
			kube.NewPermanentEnvironment("staging"),
			kube.NewPermanentEnvironment("production"),
			kube.NewPreviewEnvironment(teamNs + "-jstrachan-demo96-pr-1"),
			kube.NewPreviewEnvironment(teamNs + "-jstrachan-another-pr-3"),
			envRoleBinding,
		},
	)

	err := o.Run()
	assert.NoError(t, err)

	nsNames := []string{teamNs, teamNs + "-staging", teamNs + "-production", teamNs + "-preview-jx-jstrachan-demo96-pr-1", teamNs + "-preview-jx-jstrachan-another-pr-3"}

	for _, ns := range nsNames {
		roleBinding, err := o.KubeClient.RbacV1().RoleBindings(ns).Get(roleBindingName, metav1.GetOptions{})
		assert.NoError(t, err, "Failed to find RoleBinding in namespace %s for name %s", ns, roleBindingName)

		if roleBinding != nil && err == nil {
			assert.Equal(t, envRoleBinding.Spec.RoleRef, roleBinding.RoleRef,
				"RoleBinding.RoleRef for name %s in namespace %s", roleBindingName, ns)
		}

		r, err := o.KubeClient.RbacV1().Roles(ns).Get(roleName, metav1.GetOptions{})
		assert.NoError(t, err, "Failed to find Role in namespace %s for name %s", ns, roleName)

		if r != nil && err == nil {
			assert.Equal(t, role.Rules, r.Rules,
				"Role.Rules for name %s in namespace %s", roleBindingName, ns)
		}
		if util.StringMatchesPattern(ns, teamNs) {
			envRoleBindings, err := o.JxClient.JenkinsV1().EnvironmentRoleBindings(ns).Get(roleName, metav1.GetOptions{})
			if err != nil {
				assert.NotNil(t, envRoleBindings, "No EnvironmentRoleBinding called %s in namespace %s", roleName, ns)
			}
		}
	}

	if testhelpers.IsDebugLog() {
		namespaces, err := o.KubeClient.CoreV1().Namespaces().List(metav1.ListOptions{})
		assert.NoError(t, err)
		if err == nil {
			for _, ns := range namespaces.Items {
				testhelpers.Debugf("Has namespace %s\n", ns.Name)
			}
		}
	}

	// now lets add new user to the EnvironmentRoleBinding
	newUserKind := "ServiceAccount"
	newUser := "cheese"
	envRoleBinding, err = o.JxClient.JenkinsV1().EnvironmentRoleBindings(teamNs).Get(roleBindingName, metav1.GetOptions{})
	require.NoError(t, err, "Loading EnvironmentRoleBinding in ns %s with name %s", teamNs, roleBindingName)
	envRoleBinding.Spec.Subjects = append(envRoleBinding.Spec.Subjects, rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      newUser,
		Namespace: teamNs,
	})

	envRoleBinding, err = o.JxClient.JenkinsV1().EnvironmentRoleBindings(teamNs).PatchUpdate(envRoleBinding)
	require.NoError(t, err, "Updating EnvironmentRoleBinding in ns %s with name %s", teamNs, roleBindingName)

	// now lets simulate the watch...
	err = o.UpsertEnvironmentRoleBinding(envRoleBinding)
	require.NoError(t, err, "Failed to respond to updated EnvironmentRoleBinding in ns %s with name %s", teamNs, roleBindingName)

	AssertRoleBindingsInEnvironmentsContainsSubject(t, o.KubeClient, nsNames, roleBindingName, newUserKind, teamNs, newUser)

	message := fmt.Sprintf("For EnvironmentRoleBinding in namespace %s for name %s", teamNs, roleBindingName)

	// lets add a new preview environment
	newEnv := kube.NewPreviewEnvironment(teamNs + "-jstrachan-newthingy-pr-1")
	newPreviewNS := newEnv.Spec.Namespace
	_, err = o.JxClient.JenkinsV1().Environments(teamNs).Create(newEnv)
	require.NoError(t, err, "Failed to create an Environment %s in ns %s", newPreviewNS, teamNs)

	log.Logger().Infof("Created Preview Environment %s", newPreviewNS)

	// now lets simulate the watch...
	_ = o.UpsertEnvironmentRoleBinding(envRoleBinding)

	nsNames = append(nsNames, newPreviewNS)
	AssertRoleBindingsInEnvironmentsContainsSubject(t, o.KubeClient, nsNames, roleBindingName, newUserKind, teamNs, newUser)

	// now lets remove the user...
	envRoleBinding.Spec.Subjects = AssertRemoveSubject(t, envRoleBinding.Spec.Subjects, message, newUserKind, teamNs, newUser)
	envRoleBinding, err = o.JxClient.JenkinsV1().EnvironmentRoleBindings(teamNs).PatchUpdate(envRoleBinding)
	require.NoError(t, err, "Updating EnvironmentRoleBinding in ns %s with name %s", teamNs, roleBindingName)

	// now lets simulate the watch...
	_ = o.UpsertEnvironmentRoleBinding(envRoleBinding)

	AssertRoleBindingsInEnvironmentsNotContainsSubject(t, o.KubeClient, nsNames, roleBindingName, newUserKind, teamNs, newUser)

	// lets assert that roles get updated in all the namespaces
	AssertRolesInEnvironmentsNotContainsPolicyRule(t, o.KubeClient, nsNames, roleName, "", "get", "secrets")
	role, err = o.KubeClient.RbacV1().Roles(teamNs).Get(roleName, metav1.GetOptions{})
	require.NoError(t, err, "Failed to get Role in ns %s with name %s", teamNs, roleName)

	lastIdx := len(role.Rules) - 1
	role.Rules[lastIdx].Resources = append(role.Rules[lastIdx].Resources, "secrets")
	log.Logger().Infof("Updated Role %s to be policies %#v", roleName, role.Rules)
	_, err = o.KubeClient.RbacV1().Roles(teamNs).Update(role)
	require.NoError(t, err, "Updating EnvironmentRoleBinding in ns %s with name %s", teamNs, roleBindingName)

	// now lets simulate the watch...
	_ = o.UpsertRole(role)

	AssertRolesInEnvironmentsContainsPolicyRule(t, o.KubeClient, nsNames, roleName, "", "get", "secrets")
}

// AssertRemoveSubject removes the subject from the slice of subjects for the given kind, ns, name or fails the test
func AssertRemoveSubject(t *testing.T, subjects []rbacv1.Subject, message, kind, ns, name string) []rbacv1.Subject {
	idx := -1
	for i, subject := range subjects {
		if subject.Kind == kind && subject.Namespace == ns && subject.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		assert.Fail(t, "Should not contain subject (%s,%s,%s) for %s - has subjects %#v", kind, ns, name, message, subjects)
		return subjects
	}
	return append(subjects[0:idx], subjects[idx+1:]...)
}

// AssertRoleBindingsInEnvironmentsContainsSubject asserts that all the environments contain a role binding of the given name which contains the given subject
func AssertRoleBindingsInEnvironmentsContainsSubject(t *testing.T, kubeClient kubernetes.Interface, nsNames []string, roleBindingName, kind, teamNs, newUser string) {
	for _, ns := range nsNames {
		roleBinding, err := kubeClient.RbacV1().RoleBindings(ns).Get(roleBindingName, metav1.GetOptions{})
		require.NoError(t, err, "Failed to find RoleBinding in namespace %s for name %s", ns, roleBindingName)
		require.NotNil(t, roleBinding, "Failed to find RoleBinding in namespace %s for name %s", ns, roleBindingName)

		messsage := fmt.Sprintf("RoleBinding in namespace %s for name %s", ns, roleBindingName)
		AssertContainsSubject(t, roleBinding.Subjects, messsage, kind, teamNs, newUser)
	}
}

// AssertRoleBindingsInEnvironmentsNotContainsSubject asserts that all the environments do not contain a role binding of the given name which contains the given subject
func AssertRoleBindingsInEnvironmentsNotContainsSubject(t *testing.T, kubeClient kubernetes.Interface, nsNames []string, roleBindingName, kind, teamNs, newUser string) {
	for _, ns := range nsNames {
		roleBinding, err := kubeClient.RbacV1().RoleBindings(ns).Get(roleBindingName, metav1.GetOptions{})
		require.NoError(t, err, "Failed to find RoleBinding in namespace %s for name %s", ns, roleBindingName)
		require.NotNil(t, roleBinding, "Failed to find RoleBinding in namespace %s for name %s", ns, roleBindingName)

		messsage := fmt.Sprintf("RoleBinding in namespace %s for name %s", ns, roleBindingName)
		AssertNotContainsSubject(t, roleBinding.Subjects, messsage, kind, teamNs, newUser)
	}
}

// AssertContainsSubject asserts that the given array of subjects contains the given kind, namespace and name subject
func AssertContainsSubject(t *testing.T, subjects []rbacv1.Subject, message, kind, ns, name string) bool {
	for _, subject := range subjects {
		if subject.Kind == kind && subject.Namespace == ns && subject.Name == name {
			return true
		}
	}
	log.Logger().Warnf("Does not contain Subject: (%s,%s,%s) for %s - has subjects %#v", kind, ns, name, message, subjects)
	return assert.Fail(t, "Does not contain Subject: (%s,%s,%s) for %s - has subjects %#v", kind, ns, name, message, subjects)
}

// AssertNotContainsSubject asserts that the given array of subjects contains the given kind, namespace and name subject
func AssertNotContainsSubject(t *testing.T, subjects []rbacv1.Subject, message, kind, ns, name string) bool {
	for _, subject := range subjects {
		if subject.Kind == kind && subject.Namespace == ns && subject.Name == name {
			log.Logger().Warnf("Should not contain Subject (%s,%s,%s) for %s - has subjects %#v", kind, ns, name, message, subjects)
			return assert.Fail(t, "Should not contain Subject (%s,%s,%s) for %s - has subjects %#v", kind, ns, name, message, subjects)
		}
	}
	return true
}

// AssertRolesInEnvironmentsContainsPolicyRule asserts that all the environments contain a Role of the given name which contains the given policy rule
func AssertRolesInEnvironmentsContainsPolicyRule(t *testing.T, kubeClient kubernetes.Interface, nsNames []string, roleName, apiGroup, verb, resource string) {
	for _, ns := range nsNames {
		role, err := kubeClient.RbacV1().Roles(ns).Get(roleName, metav1.GetOptions{})
		require.NoError(t, err, "Failed to find Role in namespace %s for name %s", ns, roleName)
		require.NotNil(t, role, "Failed to find Role in namespace %s for name %s", ns, roleName)

		messsage := fmt.Sprintf("Role in namespace %s for name %s", ns, roleName)
		AssertContainsPolicyRule(t, role.Rules, messsage, apiGroup, verb, resource)
	}
}

// AssertRolesInEnvironmentsNotContainsPolicyRule asserts that all the environments do not contain a Role of the given name which contains the given policy rule
func AssertRolesInEnvironmentsNotContainsPolicyRule(t *testing.T, kubeClient kubernetes.Interface, nsNames []string, roleName, apiGroup, verb, resource string) {
	for _, ns := range nsNames {
		role, err := kubeClient.RbacV1().Roles(ns).Get(roleName, metav1.GetOptions{})
		require.NoError(t, err, "Failed to find RoleBinding in namespace %s for name %s", ns, roleName)
		require.NotNil(t, role, "Failed to find RoleBinding in namespace %s for name %s", ns, roleName)

		messsage := fmt.Sprintf("Role in namespace %s for name %s", ns, roleName)
		AssertNotContainsPolicyRule(t, role.Rules, messsage, apiGroup, verb, resource)
	}
}

// AssertContainsPolicyRule asserts that the given array of policy rules contains the given apiGroup, verb and resource subject
func AssertContainsPolicyRule(t *testing.T, rules []rbacv1.PolicyRule, message, apiGroup, verb, resource string) bool {
	for _, rule := range rules {
		if util.StringArrayIndex(rule.APIGroups, apiGroup) >= 0 && util.StringArrayIndex(rule.Verbs, verb) >= 0 && util.StringArrayIndex(rule.Resources, resource) >= 0 {
			return true
		}
	}
	log.Logger().Warnf("Does not contain PolicyRule: (%s,%s,%s) for %s - has rules %#v", apiGroup, verb, resource, message, rules)
	return assert.Fail(t, "Does not contain PolicyRule: (%s,%s,%s) for %s - has rules %#v", apiGroup, verb, resource, message, rules)
}

// AssertNotContainsPolicyRule asserts that the given array of policy rules contains the given apiGroup, verb and resource subject
func AssertNotContainsPolicyRule(t *testing.T, rules []rbacv1.PolicyRule, message, apiGroup, verb, resource string) bool {
	for _, rule := range rules {
		if util.StringArrayIndex(rule.APIGroups, apiGroup) >= 0 && util.StringArrayIndex(rule.Verbs, verb) >= 0 && util.StringArrayIndex(rule.Resources, resource) >= 0 {
			log.Logger().Warnf("Should not contain PolicyRule (%s,%s,%s) for %s - has rules %#v", apiGroup, verb, resource, message, rules)
			return assert.Fail(t, "Should not contain PolicyRule (%s,%s,%s) for %s - has rules %#v", apiGroup, verb, resource, message, rules)
		}
	}
	return true
}

package controller

import (
	"os"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/client-go/rest"

	"github.com/jenkins-x/jx-role-controller/pkg/kube"
	"github.com/jenkins-x/jx-role-controller/pkg/util"
	"github.com/pkg/errors"

	"strings"

	"github.com/jenkins-x/jx-logging/pkg/log"
	"k8s.io/client-go/kubernetes"
	kv1 "k8s.io/client-go/kubernetes/typed/rbac/v1"

	v1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-api/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-kube-client/pkg/kubeclient"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

// RoleOptions the command line options
type RoleOptions struct {
	JxClient   versioned.Interface
	KubeClient kubernetes.Interface
	kubeConfig *rest.Config
	NoWatch    bool
	TeamNs     string

	Roles           map[string]*rbacv1.Role
	EnvRoleBindings map[string]*v1.EnvironmentRoleBinding
}

const (
	blankSting = ""
	// expecting values: "true" || "yes"
	watchEnvVar             = "JX_CONTROLLER_NO_WATCH"
	roles                   = "roles"
	environments            = "environments"
	environmentrolebindings = "environmentrolebindings"
)

func NewRoleController() (*RoleOptions, error) {

	namespace, err := kubeclient.CurrentNamespace()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	kubeClient, kubeConfig, err := kube.NewClientAndConfig()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	JxClient, err := versioned.NewForConfig(kubeConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	roleController := &RoleOptions{
		JxClient:   JxClient,
		KubeClient: kubeClient,
		kubeConfig: kubeConfig,
		TeamNs:     namespace,
	}

	if os.Getenv(watchEnvVar) != "" {
		roleController.NoWatch = util.EnvVarBoolean(os.Getenv(watchEnvVar))
	}

	return roleController, nil
}

func (o *RoleOptions) Run() error {

	if !o.NoWatch {
		o.WatchRoles()
		o.WatchEnvironmentRoleBindings()
		o.WatchEnvironments()
	}

	var roles, err = o.KubeClient.RbacV1().Roles(o.TeamNs).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for idx := 0; idx < len(roles.Items); idx++ {
		err = o.UpsertRole(&roles.Items[idx])
		if err != nil {
			return errors.Wrap(err, "upserting role")
		}
	}
	bindings, err := o.JxClient.JenkinsV1().EnvironmentRoleBindings(o.TeamNs).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for i := range bindings.Items {
		err = o.UpsertEnvironmentRoleBinding(&bindings.Items[i])
		if err != nil {
			return errors.Wrap(err, "upsert environment role binding resource")
		}
	}
	envList, err := o.JxClient.JenkinsV1().Environments(o.TeamNs).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for idx := 0; idx < len(envList.Items); idx++ {
		err = o.upsertEnvironment(&envList.Items[idx])
		if err != nil {
			return err
		}
	}
	o.upsertRoleIntoEnvRole()
	return nil
}

func (o *RoleOptions) watcher(resource string, obj runtime.Object, addFunc, deleteFunc func(obj interface{}), updateFunc func(oldObj, newObj interface{})) {
	client := o.JxClient.JenkinsV1().RESTClient()
	if resource == roles {
		client = o.KubeClient.RbacV1().RESTClient()
	}
	log.Logger().Infof("starting watcher for %s resource", resource)
	listWatch := cache.NewListWatchFromClient(client, resource, o.TeamNs, fields.Everything())
	kube.SortListWatchByName(listWatch)
	_, controller := cache.NewInformer(
		listWatch,
		obj,
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    addFunc,
			UpdateFunc: updateFunc,
			DeleteFunc: deleteFunc,
		},
	)

	stop := make(chan struct{})

	log.Logger().Infof("starting controller for %s watcher", resource)
	go controller.Run(stop)

	// Wait forever
	select {}
}

func (o *RoleOptions) WatchRoles() {
	role := &rbacv1.Role{}
	o.watcher(roles, role,
		func(obj interface{}) {
			o.onRole(nil, obj)
		},
		func(obj interface{}) {
			o.onRole(obj, nil)
		},
		func(oldObj, newObj interface{}) {
			o.onRole(oldObj, newObj)
		},
	)
}

func (o *RoleOptions) WatchEnvironmentRoleBindings() {
	environmentRoleBinding := &v1.EnvironmentRoleBinding{}
	o.watcher(environmentrolebindings, environmentRoleBinding,
		func(obj interface{}) {
			o.onEnvironmentRoleBinding(nil, obj)
		},
		func(obj interface{}) {
			o.onEnvironmentRoleBinding(obj, nil)
		},
		func(oldObj, newObj interface{}) {
			o.onEnvironmentRoleBinding(oldObj, newObj)
		},
	)
}

func (o *RoleOptions) WatchEnvironments() {
	environment := &v1.Environment{}
	o.watcher(environments, environment,
		func(obj interface{}) {
			o.onEnvironment(nil, obj)
		},
		func(obj interface{}) {
			o.onEnvironment(obj, nil)
		},
		func(oldObj, newObj interface{}) {
			o.onEnvironment(oldObj, newObj)
		},
	)
}

func (o *RoleOptions) onEnvironment(oldObj, newObj interface{}) {
	var newEnv *v1.Environment
	if newObj != nil {
		newEnv = newObj.(*v1.Environment)
	}
	if oldObj != nil {
		oldEnv := oldObj.(*v1.Environment)
		if oldEnv != nil {
			if newEnv == nil || newEnv.Spec.Namespace != oldEnv.Spec.Namespace {
				o.removeEnvironmentRoleBinding(oldEnv)
			}
		}
	}
	if newEnv != nil {
		err := o.upsertEnvironment(newEnv)
		if err != nil {
			log.Logger().Warnf("Failed to upsert role bindings for environment %s: %s", newEnv.Name, err)
		}
	}
}

func (o *RoleOptions) upsertEnvironment(env *v1.Environment) error {
	log.Logger().Infof("upserting environment %s", env.Name)
	var errorMap []error
	ns := env.Spec.Namespace
	if ns != "" {
		for _, binding := range o.EnvRoleBindings {
			err := o.upsertEnvironmentRoleBindingRolesInEnvironments(env, binding, ns)
			if err != nil {
				errorMap = append(errorMap, err)
			}

		}
	}
	return util.CombineErrors(errorMap...)
}

// upsertEnvironmentRoleBindingRolesInEnvironments for the given environment and environment role binding lets update any role or role bindings if required
func (o *RoleOptions) upsertEnvironmentRoleBindingRolesInEnvironments(env *v1.Environment, binding *v1.EnvironmentRoleBinding, ns string) error {
	log.Logger().Infof("upserting environment role binding roles in environments in %s namespace", ns)
	var errorMap []error
	if kube.EnvironmentMatchesAny(env, binding.Spec.Environments) {
		var err error
		if ns != o.TeamNs {
			roleName := binding.Spec.RoleRef.Name
			role := o.Roles[roleName]
			if role == nil {
				log.Logger().Warnf("Cannot find role %s in namespace %s", roleName, o.TeamNs)
			} else {
				roles := o.KubeClient.RbacV1().Roles(ns)
				err = o.updateOrCreateRole(roles, role, roleName, ns)
			}
		}
		if err != nil {
			log.Logger().Warnf("Failed: %s", err)
			errorMap = append(errorMap, err)
		}

		bindingName := binding.Name
		roleBindings := o.KubeClient.RbacV1().RoleBindings(ns)
		var old *rbacv1.RoleBinding
		old, err = roleBindings.Get(bindingName, metav1.GetOptions{})
		if err == nil && old != nil {
			// lets update it
			changed := false

			if !reflect.DeepEqual(old.RoleRef, binding.Spec.RoleRef) {
				old.RoleRef = binding.Spec.RoleRef
				changed = true
			}
			if !reflect.DeepEqual(old.Subjects, binding.Spec.Subjects) {
				old.Subjects = binding.Spec.Subjects
				changed = true
			}
			if changed {
				log.Logger().Infof("Updating RoleBinding %s in namespace %s", bindingName, ns)
				_, err = roleBindings.Update(old)
			}
		} else {
			log.Logger().Infof("Creating RoleBinding %s in namespace %s", bindingName, ns)
			newBinding := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: bindingName,
					Labels: map[string]string{
						kube.LabelCreatedBy: kube.ValueCreatedByJX,
						kube.LabelTeam:      o.TeamNs,
					},
				},
				Subjects: binding.Spec.Subjects,
				RoleRef:  binding.Spec.RoleRef,
			}
			_, err = roleBindings.Create(newBinding)
		}
		if err != nil {
			log.Logger().Warnf("Failed: %s", err)
			errorMap = append(errorMap, err)
		}
	}
	return util.CombineErrors(errorMap...)
}

func (o *RoleOptions) updateOrCreateRole(roles kv1.RoleInterface, role *rbacv1.Role, roleName, namespace string) error {
	oldRole, err := roles.Get(roleName, metav1.GetOptions{})
	log.Logger().Infof("updating or creating role %s in namespace %s", roleName, namespace)
	if err == nil && oldRole != nil {
		// lets update it
		changed := false
		if !reflect.DeepEqual(oldRole.Rules, role.Rules) {
			oldRole.Rules = role.Rules
			changed = true
		}
		if changed {
			log.Logger().Infof("Updating Role %s in namespace %s", roleName, namespace)
			_, err = roles.Update(oldRole)
		}
	} else {
		log.Logger().Infof("Creating Role %s in namespace %s", roleName, namespace)
		newRole := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name: roleName,
				Labels: map[string]string{
					kube.LabelCreatedBy: kube.ValueCreatedByJX,
					kube.LabelTeam:      o.TeamNs,
				},
			},
			Rules: role.Rules,
		}
		_, err = roles.Create(newRole)
	}
	return err
}

func (o *RoleOptions) removeEnvironmentRoleBinding(env *v1.Environment) {
	log.Logger().Infof("removing environment role binding for %s", env.Name)
	ns := env.Spec.Namespace
	if ns != "" {
		for _, binding := range o.EnvRoleBindings {
			if kube.EnvironmentMatchesAny(env, binding.Spec.Environments) {
				err := o.KubeClient.RbacV1().RoleBindings(ns).Delete(binding.Name, nil)
				if err != nil {
					log.Logger().Errorf("error deleting role binding from env: %s", binding.Name)
				}
			}
		}
	}
}

func (o *RoleOptions) onEnvironmentRoleBinding(oldObj, newObj interface{}) {
	if o.EnvRoleBindings == nil {
		o.EnvRoleBindings = map[string]*v1.EnvironmentRoleBinding{}
	}
	if oldObj != nil {
		oldEnv := oldObj.(*v1.EnvironmentRoleBinding)
		if oldEnv != nil {
			delete(o.EnvRoleBindings, oldEnv.Name)
		}
	}
	if newObj != nil {
		newEnv := newObj.(*v1.EnvironmentRoleBinding)
		err := o.UpsertEnvironmentRoleBinding(newEnv)
		if err != nil {
			log.Logger().Warnf("when upserting environment role binding %v", err)
		}
	}
}

// UpsertEnvironmentRoleBinding processes an insert/update of the EnvironmentRoleBinding resource
// its public so that we can make testing easier
func (o *RoleOptions) UpsertEnvironmentRoleBinding(newEnv *v1.EnvironmentRoleBinding) error {
	log.Logger().Info("upserting environment role binding")
	if newEnv != nil {
		if o.EnvRoleBindings == nil {
			o.EnvRoleBindings = map[string]*v1.EnvironmentRoleBinding{}
		}
		o.EnvRoleBindings[newEnv.Name] = newEnv
	}

	// now lets update any roles in any environment we may need to change
	envList, err := o.JxClient.JenkinsV1().Environments(o.TeamNs).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	var errorMap []error
	for idx := 0; idx < len(envList.Items); idx++ {
		env := &envList.Items[idx]
		err = o.upsertEnvironmentRoleBindingRolesInEnvironments(env, newEnv, env.Spec.Namespace)
		if err != nil {
			errorMap = append(errorMap, err)
		}
	}
	return util.CombineErrors(errorMap...)
}

func (o *RoleOptions) onRole(oldObj, newObj interface{}) {
	if o.Roles == nil {
		o.Roles = map[string]*rbacv1.Role{}
	}
	if oldObj != nil {
		oldRole := oldObj.(*rbacv1.Role)
		if oldRole != nil {
			delete(o.Roles, oldRole.Name)
		}
	}
	if newObj != nil {
		newRole := newObj.(*rbacv1.Role)
		err := o.UpsertRole(newRole)
		if err != nil {
			log.Logger().Errorf("when upserting role: %s", newRole.Name)
		}
	}
}

// UpsertRole processes the insert/update of a Role
// this function is public for easier testing
func (o *RoleOptions) UpsertRole(newRole *rbacv1.Role) error {
	log.Logger().Info("upserting role")
	if newRole == nil {
		return nil
	}
	if o.Roles == nil {
		o.Roles = map[string]*rbacv1.Role{}
	}
	o.Roles[newRole.Name] = newRole

	if newRole.Labels == nil || newRole.Labels[kube.LabelKind] != kube.ValueKindEnvironmentRole {
		return nil
	}

	// now lets update any roles in any environment we may need to change
	envList, err := o.JxClient.JenkinsV1().Environments(o.TeamNs).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	var errorMap []error
	for idx := 0; idx < len(envList.Items); idx++ {
		err = o.upsertRoleInEnvironments(newRole, envList.Items[idx].Spec.Namespace)
		if err != nil {
			errorMap = append(errorMap, err)
		}
	}
	return util.CombineErrors(errorMap...)
}

// upsertRoleInEnvironments updates the Role in the team environment in the other environment namespaces if it has changed
func (o *RoleOptions) upsertRoleInEnvironments(role *rbacv1.Role, ns string) error {
	log.Logger().Infof("upserting role into environment for %s in %s namespace", role.Name, ns)
	if ns == o.TeamNs {
		return nil
	}
	var err error
	roles := o.KubeClient.RbacV1().Roles(ns)
	err = o.updateOrCreateRole(roles, role, role.Name, ns)
	return err
}

func (o *RoleOptions) upsertRoleIntoEnvRole() {
	log.Logger().Info("upserting role into environment role")
	foundRole := 0
	for _, roleValue := range o.Roles {
		for labelK, labelV := range roleValue.Labels {
			if util.StringMatchesPattern(labelK, kube.LabelKind) && util.StringMatchesPattern(labelV, kube.ValueKindEnvironmentRole) {
				for _, envRoleValue := range o.EnvRoleBindings {
					if util.StringMatchesPattern(strings.Trim(roleValue.GetName(), blankSting), strings.Trim(envRoleValue.Spec.RoleRef.Name, blankSting)) {
						foundRole = 1
						break
					}
				}
				if foundRole == 0 {
					log.Logger().Infof("Environment binding doesn't exist for role %s , creating it.", util.ColorInfo(roleValue.GetName()))
					newSubject := rbacv1.Subject{
						Name:      roleValue.GetName(),
						Kind:      kube.ValueKindEnvironmentRole,
						Namespace: o.TeamNs,
					}
					newEnvRoleBinding := &v1.EnvironmentRoleBinding{
						ObjectMeta: metav1.ObjectMeta{
							Name:      roleValue.GetName(),
							Namespace: o.TeamNs,
						},
						Spec: v1.EnvironmentRoleBindingSpec{
							Subjects: []rbacv1.Subject{
								newSubject,
							},
						},
					}
					_, err := o.JxClient.JenkinsV1().EnvironmentRoleBindings(o.TeamNs).Create(newEnvRoleBinding)
					if err != nil {
						log.Logger().Errorf("when upserting role into environment role: %s, with error: %s", newEnvRoleBinding.Name, err)
					}
				}
			}
		}

	}
}

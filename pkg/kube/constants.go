package kube

const (
	// ValueCreatedByJX for resources created by the Jenkins X CLI
	ValueCreatedByJX = "jx"

	// LabelCreatedBy indicates the service that created this resource
	LabelCreatedBy = "jenkins.io/created-by"

	// LabelTeam indicates the team name an environment belongs to
	LabelTeam = "team"

	// ValueKindEnvironmentRole to indicate a Role which maps to an EnvironmentRoleBinding
	ValueKindEnvironmentRole = "EnvironmentRole"

	// LabelKind to indicate the kind of auth, such as Git or Issue
	LabelKind = "jenkins.io/kind"
)

package rbac

type GroupRoleBinding struct {
	Group Group  `json:"group"`
	Roles []Role `json:"roles,omitempty"`
}

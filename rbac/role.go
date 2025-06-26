package rbac

type Role struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions,omitempty"`
}

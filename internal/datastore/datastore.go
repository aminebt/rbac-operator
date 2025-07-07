package datastore

import (
	"fmt"

	"github.com/aminebt/rbac-operator/rbac"
)

type DataStore interface {
	GetGroup(groupName string) (rbac.Group, int, error)
	CreateGroup(gr rbac.Group) (int, error)
	DeleteGroup(groupName string) (rbac.Group, error)

	GetRole(roleName string) (rbac.Role, error)
	CreateUpdateRole(roleName string) (rbac.Role, error)
	DeleteRole(roleName string) (rbac.Role, error)

	GetGRBinding(roleName string) (rbac.GroupRoleBinding, error)
	CreateGRBinding(roleName string) (rbac.GroupRoleBinding, error)
	DeleteGRBinding(roleName string) (rbac.GroupRoleBinding, error)

	CleanUp()
}

func NewDataStore(backend string) (DataStore, error) {
	switch backend {
	case "kubernetes":
		kds, err := NewKubernetesDataStore()
		return kds, err
	case "postgres":
		pgds, err := NewPostgresDataStore()
		return pgds, err
	default:
		return nil, fmt.Errorf("unsupported backend: %s", backend)
	}
}

package datastore

import (
	"fmt"

	"github.com/aminebt/rbac-operator/internal/env"
	"github.com/aminebt/rbac-operator/rbac"
)

type DataStore interface {
	GetGroup(env env.Environment, groupName string) (rbac.Group, int, error)
	CreateGroup(env env.Environment, gr rbac.Group) (int, error)
	DeleteGroup(env env.Environment, groupName string) (int, error)

	GetRole(env env.Environment, roleName string) (rbac.Role, int, error)
	CreateRole(env env.Environment, role rbac.Role) (int, error)
	DeleteRole(env env.Environment, roleName string) (int, error)

	GetGRBinding(env env.Environment, roleName string) (rbac.GroupRoleBinding, error)
	CreateGRBinding(env env.Environment, roleName string) (rbac.GroupRoleBinding, error)
	DeleteGRBinding(env env.Environment, roleName string) (rbac.GroupRoleBinding, error)

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

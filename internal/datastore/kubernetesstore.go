package datastore

import (
	"log/slog"
	"os"

	"github.com/aminebt/rbac-operator/internal/env"
	"github.com/aminebt/rbac-operator/rbac"
)

type KubernetesDataStore struct {
	logger *slog.Logger
}

func NewKubernetesDataStore() (DataStore, error) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	kds := &KubernetesDataStore{
		logger: logger,
	}

	return kds, nil
}

func (kds *KubernetesDataStore) CleanUp() {
	kds.logger.Info("Nothing to clean up for Kubernetes DataStore")
}

func (kds *KubernetesDataStore) GetGroup(env env.Environment, groupName string) (rbac.Group, int, error) {
	return rbac.Group{}, 0, nil
}

func (kds *KubernetesDataStore) CreateGroup(env env.Environment, gr rbac.Group) (int, error) {
	return 0, nil
}

func (kds *KubernetesDataStore) DeleteGroup(env env.Environment, groupName string) (int, error) {
	return 0, nil
}

// Role methods
func (kds *KubernetesDataStore) GetRole(env env.Environment, roleName string) (rbac.Role, int, error) {
	return rbac.Role{}, 0, nil
}

func (kds *KubernetesDataStore) CreateRole(env env.Environment, role rbac.Role) (int, error) {
	//TBD
	return 0, nil
}

func (kds *KubernetesDataStore) DeleteRole(env env.Environment, roleName string) (int, error) {
	//TBD
	return 0, nil
}

// GRBinding
func (kds *KubernetesDataStore) GetGRBinding(env env.Environment, roleName string) (rbac.GroupRoleBinding, error) {
	return rbac.GroupRoleBinding{}, nil
}

func (kds *KubernetesDataStore) CreateGRBinding(env env.Environment, roleName string) (rbac.GroupRoleBinding, error) {
	return rbac.GroupRoleBinding{}, nil
}

func (kds *KubernetesDataStore) DeleteGRBinding(env env.Environment, roleName string) (rbac.GroupRoleBinding, error) {
	return rbac.GroupRoleBinding{}, nil
}

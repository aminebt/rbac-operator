/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"github.com/aminebt/rbac-operator/internal/controller/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type Role struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions,omitempty"`
}

// GridOSRoleSpec defines the desired state of GridOSRole.
type GridOSRoleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of GridOSRole. Edit gridosrole_types.go to remove/update
	Permissions []string `json:"permissions,omitempty"`
}

// GridOSRoleStatus defines the observed state of GridOSRole.
type GridOSRoleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	BoundGroups []string          `json:"groups,omitempty"`
	Bindings    map[string]string `json:"bindings,omitempty"`
	Phase       StatusPhase       `json:"phase"`
	LastMessage string            `json:"lastMessage"`
}

func (s *GridOSRoleStatus) Update(phase StatusPhase, msg string, err error) {
	if err != nil {
		msg = msg + ": " + err.Error()
	}

	s.Phase = phase
	s.LastMessage = msg
}

func (s *GridOSRoleStatus) UpdateBinding(binding, group string, deleteBinding bool) {
	if s.Bindings == nil {
		s.Bindings = make(map[string]string)
	}
	if s.BoundGroups == nil {
		s.BoundGroups = make([]string, 0)
	}

	// if group not found (or other error fetching group), remove that binding
	if deleteBinding {
		//TO DO - add debug level log
		delete(s.Bindings, binding)
	} else {
		s.Bindings[binding] = group
	}

	s.BoundGroups = utils.UniqueValues(s.Bindings)
}

// func (s *GridOSRoleStatus) DeleteBinding(binding string) {
// 	if s.Bindings == nil {
// 		s.Bindings = make(map[string]string)
// 	}
// 	if s.BoundGroups == nil {
// 		s.BoundGroups = make([]string, 0)
// 	}
// 	delete(s.Bindings, binding)
// 	s.BoundGroups = utils.UniqueValues(s.Bindings)
// }

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// GridOSRole is the Schema for the gridosroles API.
type GridOSRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GridOSRoleSpec   `json:"spec,omitempty"`
	Status GridOSRoleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GridOSRoleList contains a list of GridOSRole.
type GridOSRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GridOSRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GridOSRole{}, &GridOSRoleList{})
}

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

// GridOSGroupStatus defines the observed state of GridOSGroup.
type GridOSGroupStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Bindings    map[string][]string `json:"bindings,omitempty"`
	BoundRoles  []string            `json:"roles,omitempty"`
	Phase       StatusPhase         `json:"phase"`
	LastMessage string              `json:"lastMessage"`
}

func (s *GridOSGroupStatus) Update(phase StatusPhase, msg string, err error) {
	if err != nil {
		msg = msg + ": " + err.Error()
	}

	s.Phase = phase
	s.LastMessage = msg
}

func (s *GridOSGroupStatus) UpdateBindings(binding string, roles []string) {
	if s.Bindings == nil {
		s.Bindings = make(map[string][]string)
	}
	if s.BoundRoles == nil {
		s.BoundRoles = make([]string, 0)
	}
	if len(roles) > 0 {
		s.Bindings[binding] = roles
	} else {
		delete(s.Bindings, binding)
	}

	s.BoundRoles = utils.UniqueArrayValues(s.Bindings)
}

func (s *GridOSGroupStatus) DeleteBindings(binding string) {
	delete(s.Bindings, binding)
	s.BoundRoles = utils.UniqueArrayValues(s.Bindings)
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// GridOSGroup is the Schema for the gridosgroups API.
type GridOSGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	//Spec   GridOSGroupSpec   `json:"spec,omitempty"`
	Status GridOSGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GridOSGroupList contains a list of GridOSGroup.
type GridOSGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GridOSGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GridOSGroup{}, &GridOSGroupList{})
}

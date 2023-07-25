/*
 * Copyright 2023 Nikolai Rodionov (allanger)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1beta1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DbUserSpec defines the desired state of DbUser
type DbUserSpec struct {
	// DatabaseRef should contain a name of a Database to create a user there
	// Database should be in the same namespace with the user
	DatabaseRef string `json:"databaseRef"`
	// AccessType that should be given to a user
	// Currently only readOnly and readWrite are supported by the operator
	AccessType string `json:"accessType"`
	// SecretName name that should be used to save user's credentials
	SecretName string `json:"secretName"`
}

// DbUserStatus defines the observed state of DbUser
type DbUserStatus struct {
	Status       bool   `json:"status"`
	DatabaseName string `json:"database"`
	Phase        string `json:"phase"`
	// It's required to let the operator update users
	Created bool `json:"created"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type=boolean,JSONPath=`.status.status`,description="current dbuser status"
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="current dbuser phase"
//+kubebuilder:printcolumn:name="DatabaseName",type=string,JSONPath=`.spec.databaseRef`,description="To which database user should have access"
//+kubebuilder:printcolumn:name="AccessType",type=string,JSONPath=`.spec.accessType`,description="A type of access the user has"
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,description="time since creation of resos¡urce"

// DbUser is the Schema for the dbusers API
type DbUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DbUserSpec   `json:"spec,omitempty"`
	Status DbUserStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DbUserList contains a list of DbUser
type DbUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DbUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DbUser{}, &DbUserList{})
}

// Access types that are supported by the operator
const (
	READONLY  = "readOnly"
	READWRITE = "readWrite"
)

// IsAccessTypeSupported returns an error if access type is not supported
func IsAccessTypeSupported(wantedAccessType string) error {
	supportedAccessTypes := []string{READONLY, READWRITE}
	for _, supportedAccessType := range supportedAccessTypes {
		if supportedAccessType == wantedAccessType {
			return nil
		}
	}
	return fmt.Errorf("the provided access type is not supported by the operator: %s - please chose one of these: %v",
		wantedAccessType,
		supportedAccessTypes,
	)
}

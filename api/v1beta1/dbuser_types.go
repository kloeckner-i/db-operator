/*
 * Copyright 2023 DB-Operator Authors
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
	"slices"

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
	// A list of additional roles that should be added to the user
	ExtraPrivileges []string    `json:"extraPrivileges,omitempty"`
	Credentials     Credentials `json:"credentials,omitempty"`
	Cleanup         bool        `json:"cleanup,omitempty"`
	// Should the user be granted to the admin user
	// For example, it should be set to true on Azure instance,
	// because the admin given by them is not a super user,
	// but should be set to false on AWS, when rds_iam extra
	// privilege is added
	// By default is set to true
	// Only applies to Postgres, doesn't have any effect on Mysql
	// TODO: Default should be false, but not to introduce breaking
	//       changes it's now set to true. It should be changed in
	//       in the next API version
	// +kubebuilder:default=true
	// +optional
	GrantToAdmin bool `json:"grantToAdmin"`
	// If specified, DB Operator will try to use an existing user to assign permissions
	// User will not be removed, when a dbuser is removed, but the permissions added by the
	// operator will be cleaned up
	ExistingUser string `json:"existingUser,omitempty"`
}

// DbUserStatus defines the observed state of DbUser
type DbUserStatus struct {
	Status       bool   `json:"status"`
	DatabaseName string `json:"database"`
	// It's required to let the operator update users
	Created         bool   `json:"created"`
	OperatorVersion string `json:"operatorVersion,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type=boolean,JSONPath=`.status.status`,description="current dbuser status"
//+kubebuilder:printcolumn:name="DatabaseName",type=string,JSONPath=`.spec.databaseRef`,description="To which database user should have access"
//+kubebuilder:printcolumn:name="AccessType",type=string,JSONPath=`.spec.accessType`,description="A type of access the user has"
//+kubebuilder:printcolumn:name="OperatorVersion",type=string,JSONPath=`.status.operatorVersion`,description="db-operator version of last full reconcile"
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,description="time since creation of resource"

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
	if slices.Contains(supportedAccessTypes, wantedAccessType) {
		return nil
	}
	return fmt.Errorf("the provided access type is not supported by the operator: %s - please choose one of these: %v",
		wantedAccessType,
		supportedAccessTypes,
	)
}

// DbUsers don't have cleanup feature implemented
func (dbu *DbUser) IsCleanup() bool {
	return dbu.Spec.Cleanup
}

func (dbu *DbUser) IsDeleted() bool {
	return dbu.GetDeletionTimestamp() != nil
}

func (dbu *DbUser) GetSecretName() string {
	return dbu.Spec.SecretName
}

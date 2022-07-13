/*
 * Copyright 2021 kloeckner.i GmbH
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

package v1alpha1

import (
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DatabaseSpec defines the desired state of Database
type DatabaseSpec struct {
	SecretName        string         `json:"secretName"`
	Instance          string         `json:"instance"`
	DeletionProtected bool           `json:"deletionProtected"`
	Backup            DatabaseBackup `json:"backup"`
	Extensions        []string       `json:"extensions,omitempty"`
	// ConnectionStringTemplate field can be used to pass a custom template for generating a db connection string.
	// These keywords can be used: Protocol, DatabaseHost, DatabasePort, UserName, Password, DatabaseName.
	// Default template looks like this:
	// "{{ .Protocol }}://{{ .UserName }}:{{ .Password }}@{{ .DatabaseHost }}:{{ .DatabasePort }}/{{ .DatabaseName }}"
	ConnectionStringTemplate string   `json:"connectionStringTemplate,omitempty"`
	Postgres                 Postgres `json:"postgres,omitempty"`
}

// Postgres struct should be used to provide resource that only applicable to postgres
type Postgres struct {
	// If set to true, the public schema will be dropped after the database creation
	DropPublicSchema bool `json:"dropPublicSchema,omitempty"`
	// Specify schemas to be created. The user created by db-operator will have all access on them.
	Schemas []string `json:"schemas,omitempty"`
}

// DatabaseStatus defines the observed state of Database
type DatabaseStatus struct {
	// Important: Run "make generate" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Phase                 string              `json:"phase"`
	Status                bool                `json:"status"`
	InstanceRef           *DbInstance         `json:"instanceRef"`
	MonitorUserSecretName string              `json:"monitorUserSecret,omitempty"`
	ProxyStatus           DatabaseProxyStatus `json:"proxyStatus,omitempty"`
	DatabaseName          string              `json:"database"`
	UserName              string              `json:"user"`
}

// DatabaseProxyStatus defines whether proxy for database is enabled or not
// if so, provide information
type DatabaseProxyStatus struct {
	Status      bool   `json:"status"`
	ServiceName string `json:"serviceName"`
	SQLPort     int32  `json:"sqlPort"`
}

// DatabaseBackup defines the desired state of backup and schedule
type DatabaseBackup struct {
	Enable bool   `json:"enable"`
	Cron   string `json:"cron"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=db
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`,description="current db phase"
//+kubebuilder:printcolumn:name="Status",type=boolean,JSONPath=`.status.status`,description="current db status"
//+kubebuilder:printcolumn:name="Protected",type=boolean,JSONPath=`.spec.deletionProtected`,description="If database is protected to not get deleted."
//+kubebuilder:printcolumn:name="DBInstance",type=string,JSONPath=`.spec.instance`,description="instance reference"
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,description="time since creation of resource"

// Database is the Schema for the databases API
type Database struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseSpec   `json:"spec,omitempty"`
	Status DatabaseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DatabaseList contains a list of Database
type DatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Database `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Database{}, &DatabaseList{})
}

// GetInstanceRef returns DbInstance pointer which used by Database
func (db *Database) GetInstanceRef() (*DbInstance, error) {
	if db.Status.InstanceRef == nil {
		return nil, errors.New("can not find instance ref")
	}
	return db.Status.InstanceRef, nil
}

// GetEngineType returns type of database engine ex) postgres or mysql
func (db *Database) GetEngineType() (string, error) {
	instance, err := db.GetInstanceRef()
	if err != nil {
		return "", err
	}

	return instance.Spec.Engine, nil
}

// GetBackendType returns type of instance infrastructure.
// Infrastructure where database is running ex) google cloud sql, generic instance
func (db *Database) GetBackendType() (string, error) {
	instance, err := db.GetInstanceRef()
	if err != nil {
		return "", err
	}

	return instance.GetBackendType()
}

// IsMonitoringEnabled returns true if monitoring is enabled in DbInstance spec.
func (db *Database) IsMonitoringEnabled() (bool, error) {
	instance, err := db.GetInstanceRef()
	if err != nil {
		return false, err
	}

	return instance.IsMonitoringEnabled(), nil
}

// AccessSecretName returns string value to define name of the secret resource for accessing instance
func (db *Database) InstanceAccessSecretName() string {
	return "dbin-" + db.Spec.Instance + "-access-secret"
}

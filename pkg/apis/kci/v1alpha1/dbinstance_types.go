package v1alpha1

import (
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DbInstanceSpec defines the desired state of DbInstance
// +k8s:openapi-gen=true
type DbInstanceSpec struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	Engine          string               `json:"engine"`
	AdminUserSecret types.NamespacedName `json:"adminSecretRef"`
	Backup          DbInstanceBackup     `json:"backup"`
	Monitoring      DbInstanceMonitoring `json:"monitoring"`
	Google          *GoogleInstance      `json:"google,omitempty"`
	Generic         *GenericInstance     `json:"generic,omitempty"`
}

// DbInstanceStatus defines the observed state of DbInstance
// +k8s:openapi-gen=true
type DbInstanceStatus struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	Phase     string            `json:"phase"`
	Status    bool              `json:"status"`
	Info      map[string]string `json:"info,omitempty"`
	Checksums map[string]string `json:"checksums,omitempty"`
}

// GoogleInstance is used when instance type is Google Cloud SQL
// and describes necessary informations to use google API to create sql instances
type GoogleInstance struct {
	InstanceName  string               `json:"instance"`
	ConfigmapName types.NamespacedName `json:"configmapRef"`
}

// GenericInstance is used when instance type is generic
// and describes necessary informations to use instance
// generic instance can be any backend, it must be reachable by described address and port
type GenericInstance struct {
	Host     string `json:"host"`
	Port     int32  `json:"port"`
	PublicIP string `json:"publicIp,omitempty"`
	// BackupHost address will be used for dumping database for backup
	// Usually slave address for master-slave setup or cluster lb address
	// If it's not defined, above Host will be used as backup host address.
	BackupHost string `json:"backupHost"`
}

// DbInstanceBackup defines name of google bucket to use for storing database dumps for backup when backup is enabled
type DbInstanceBackup struct {
	Bucket string `json:"bucket"`
}

// DbInstanceMonitoring defines if exporter
type DbInstanceMonitoring struct {
	Enabled bool `json:"enabled"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DbInstance is the Schema for the dbinstances API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=dbinstances,scope=Clustered
type DbInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DbInstanceSpec   `json:"spec,omitempty"`
	Status DbInstanceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DbInstanceList contains a list of DbInstance
type DbInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DbInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DbInstance{}, &DbInstanceList{})
}

// ValidateEngine checks if defined engine by DbInstance object is supported by db-operator
func (dbin *DbInstance) ValidateEngine() error {
	if (dbin.Spec.Engine == "mysql") || (dbin.Spec.Engine == "postgres") {
		return nil
	}

	return errors.New("not supported engine type")
}

// ValidateBackend checks if backend type of instance is defined properly
// returns error when more than one backend types are defined
// or when no backend type is defined
func (dbin *DbInstance) ValidateBackend() error {
	if (dbin.Spec.Google != nil) && (dbin.Spec.Generic != nil) {
		return errors.New("more than one instance type defined")
	}

	if (dbin.Spec.Google == nil) && (dbin.Spec.Generic == nil) {
		return errors.New("no instance type defined")
	}

	return nil
}

// GetBackendType returns type of instance infrastructure.
// Infrastructure where database is running ex) google cloud sql, generic instance
func (dbin *DbInstance) GetBackendType() (string, error) {
	err := dbin.ValidateBackend()
	if err != nil {
		return "", err
	}
	if dbin.Spec.Google != nil {
		return "google", nil
	}

	if dbin.Spec.Generic != nil {
		return "generic", nil
	}

	return "", errors.New("no backend type defined")
}

// IsMonitoringEnabled returns boolean value if monitoring is enabled for the instance
func (dbin *DbInstance) IsMonitoringEnabled() bool {
	if dbin.Spec.Monitoring.Enabled == false {
		return false
	}

	return true
}

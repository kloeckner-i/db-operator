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
	Engine           string               `json:"engine"`
	AdminUserSecret  types.NamespacedName `json:"adminSecretRef"`
	Backup           DbInstanceBackup     `json:"backup"`
	Monitoring       DbInstanceMonitoring `json:"monitoring"`
	DbInstanceSource `json:",inline"`
}

// DbInstanceSource represents the source of a instance.
// Only one of its members may be specified.
type DbInstanceSource struct {
	Google  *GoogleInstance  `json:"google,omitempty" protobuf:"bytes,1,opt,name=google"`
	Generic *GenericInstance `json:"generic,omitempty" protobuf:"bytes,2,opt,name=generic"`
	Percona *PerconaCluster  `json:"percona,omitempty" protobuf:"bytes,3,opt,name=percona"`
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

// PerconaCluster is used when instance type is percona cluster
type PerconaCluster struct {
	ServerList        []BackendServer      `json:"servers"` // hostgroup: host address
	MonitorUserSecret types.NamespacedName `json:"monitorUserSecretRef"`
}

// BackendServer defines backend database server
type BackendServer struct {
	Host          string `json:"host"`
	Port          uint16 `json:"port"`
	MaxConnection uint16 `json:"maxConn"`
	ReadOnly      bool   `json:"readonly,omitempty"`
}

// GenericInstance is used when instance type is generic
// and describes necessary informations to use instance
// generic instance can be any backend, it must be reachable by described address and port
type GenericInstance struct {
	Host     string `json:"host"`
	Port     uint16 `json:"port"`
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
	source := dbin.Spec.DbInstanceSource

	if (source.Google == nil) && (source.Generic == nil) && (source.Percona == nil) {
		return errors.New("no instance type defined")
	}

	numSources := 0

	if source.Google != nil {
		numSources++
	}

	if source.Generic != nil {
		numSources++
	}

	if source.Percona != nil {
		numSources++
	}

	if numSources > 1 {
		return errors.New("may not specify more than 1 instance type")
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

	source := dbin.Spec.DbInstanceSource

	if source.Google != nil {
		return "google", nil
	}

	if source.Generic != nil {
		return "generic", nil
	}

	if source.Percona != nil {
		return "percona", nil
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

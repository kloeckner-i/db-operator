package proxy

import (
	"bytes"
	"log"
	"strconv"
	"text/template"

	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	proxysql "github.com/kloeckner-i/db-operator/pkg/utils/proxy/proxysql"

	v1apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProxySQL for percona cluster
type ProxySQL struct {
	NamePrefix            string
	Namespace             string
	Servers               []proxysql.Backend
	UserSecretName        string
	MonitorUserSecretName string
	Engine                string
	Labels                map[string]string
	configCheckSum        string
}

const sqlPort = 6033
const adminPort = 6032

func (ps *ProxySQL) configMapName() string {
	return ps.NamePrefix + "-proxysql-config-template"
}

func (ps *ProxySQL) serviceName() string {
	return ps.NamePrefix + "-proxysql"
}

func (ps *ProxySQL) buildService() (*v1.Service, error) {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ps.serviceName(),
			Namespace: ps.Namespace,
			Labels:    ps.Labels,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Name:     ps.Engine,
					Protocol: v1.ProtocolTCP,
					Port:     sqlPort,
				},
			},
			Selector: ps.Labels,
		},
	}, nil
}

func (ps *ProxySQL) buildDeployment() (*v1apps.Deployment, error) {
	spec, err := ps.deploymentSpec()
	if err != nil {
		return nil, err
	}

	return &v1apps.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ps.NamePrefix + "-proxysql",
			Namespace: ps.Namespace,
			Labels:    ps.Labels,
		},
		Spec: spec,
	}, nil
}

func (ps *ProxySQL) deploymentSpec() (v1apps.DeploymentSpec, error) {
	var replicas int32 = 2

	configGenContainer, err := ps.configGeneratorContainer()
	if err != nil {
		return v1apps.DeploymentSpec{}, err
	}

	proxyContainer, err := ps.proxyContainer()
	if err != nil {
		return v1apps.DeploymentSpec{}, err
	}

	volumes := []v1.Volume{
		v1.Volume{
			Name: "shared-data",
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		},
		v1.Volume{
			Name: "proxysql-config-template",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: ps.configMapName(),
					},
				},
			},
		},
		v1.Volume{
			Name: "monitoruser-secret",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: ps.MonitorUserSecretName,
					Items: []v1.KeyToPath{
						v1.KeyToPath{
							Key:  "password",
							Path: "monitoruser-password",
						},
					},
				},
			},
		},
		v1.Volume{
			Name: "user-secret",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: ps.UserSecretName,
					Items: []v1.KeyToPath{
						v1.KeyToPath{
							Key:  "PASSWORD",
							Path: "user-password",
						},
					},
				},
			},
		},
	}

	annotations := make(map[string]string)
	annotations["checksum/config"] = ps.configCheckSum

	return v1apps.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: ps.Labels,
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      ps.Labels,
				Annotations: annotations,
			},
			Spec: v1.PodSpec{
				InitContainers: []v1.Container{configGenContainer},
				Containers:     []v1.Container{proxyContainer},
				NodeSelector:   conf.Instances.Percona.ProxyConfig.NodeSelector,
				RestartPolicy:  v1.RestartPolicyAlways,
				Volumes:        volumes,
				Affinity: &v1.Affinity{
					PodAntiAffinity: podAntiAffinity(ps.Labels),
				},
			},
		},
	}, nil
}

func (ps *ProxySQL) configGeneratorContainer() (v1.Container, error) {
	return v1.Container{
		Name:            "config-generator",
		Image:           "dibi/envsubst",
		ImagePullPolicy: v1.PullIfNotPresent,
		Command:         []string{"sh", "-c", "MONITOR_PASSWORD=$(cat /run/secrets/monitoruser-password) DB_PASSWORD=$(cat /run/secrets/user-password) envsubst < /tmp/proxysql.cnf.tmpl > /mnt/proxysql.cnf"},
		Env: []v1.EnvVar{
			v1.EnvVar{
				Name: "MONITOR_USERNAME", ValueFrom: &v1.EnvVarSource{
					SecretKeyRef: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{Name: ps.MonitorUserSecretName},
						Key:                  "user",
					},
				},
			},
			v1.EnvVar{
				Name: "DB_USERNAME", ValueFrom: &v1.EnvVarSource{
					SecretKeyRef: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{Name: ps.UserSecretName},
						Key:                  "USER",
					},
				},
			},
		},
		VolumeMounts: []v1.VolumeMount{
			v1.VolumeMount{
				Name:      "proxysql-config-template",
				MountPath: "/tmp/proxysql.cnf.tmpl",
				SubPath:   "proxysql.cnf.tmpl",
			},
			v1.VolumeMount{
				Name:      "shared-data",
				MountPath: "/mnt",
			},
			v1.VolumeMount{
				Name:      "monitoruser-secret",
				MountPath: "/run/secrets/monitoruser-password",
				SubPath:   "monitoruser-password",
				ReadOnly:  true,
			},
			v1.VolumeMount{
				Name:      "user-secret",
				MountPath: "/run/secrets/user-password",
				SubPath:   "user-password",
				ReadOnly:  true,
			},
		},
	}, nil
}

func (ps *ProxySQL) proxyContainer() (v1.Container, error) {
	return v1.Container{
		Name:            "proxysql",
		Image:           conf.Instances.Percona.ProxyConfig.Image,
		ImagePullPolicy: v1.PullIfNotPresent,
		Ports: []v1.ContainerPort{
			v1.ContainerPort{
				Name:          "sql",
				ContainerPort: sqlPort,
				Protocol:      v1.ProtocolTCP,
			},
			v1.ContainerPort{
				Name:          "admin",
				ContainerPort: adminPort,
				Protocol:      v1.ProtocolTCP,
			},
		},
		VolumeMounts: []v1.VolumeMount{
			v1.VolumeMount{
				Name:      "shared-data",
				MountPath: "/etc/proxysql.cnf",
				SubPath:   "proxysql.cnf",
			},
		},
		Resources: v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("50m"),
				v1.ResourceMemory: resource.MustParse("128Mi"),
			},
		},
	}, nil
}

func (ps *ProxySQL) buildConfigMap() (*v1.ConfigMap, error) {
	configTmpl := proxysql.PerconaMysqlConfigTemplate
	t := template.Must(template.New("config").Parse(configTmpl))

	config := proxysql.Config{
		AdminPort: strconv.FormatInt(int64(adminPort), 10),
		SQLPort:   strconv.FormatInt(int64(sqlPort), 10),
		Backends:  ps.Servers,
	}

	var outputBuf bytes.Buffer
	if err := t.Execute(&outputBuf, config); err != nil {
		log.Fatal(err)
	}

	data := map[string]string{
		"proxysql.cnf.tmpl": outputBuf.String(),
	}

	ps.configCheckSum = kci.GenerateChecksum(data)

	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ps.Namespace,
			Name:      ps.configMapName(),
			Labels:    ps.Labels,
		},
		Data: data,
	}, nil
}

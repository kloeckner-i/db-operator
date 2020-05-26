package proxy

import (
	"bytes"
	"html/template"
	"log"
	"strconv"

	v1apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProxySQL for percona cluster
type ProxySQL struct {
	NamePrefix            string
	Namespace             string
	Servers               []string
	MaxConn               int16
	MonitorUserSecretName string
	Engine                string
	Port                  int32
	Labels                map[string]string
}

func (ps *ProxySQL) buildService() (*v1.Service, error) {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ps.NamePrefix + "-svc",
			Namespace: ps.Namespace,
			Labels:    ps.Labels,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Name:     ps.Engine,
					Protocol: v1.ProtocolTCP,
					Port:     ps.Port,
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
			APIVersion: "extensions/apps",
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

	proxyContainer, err := ps.proxyContainer()
	if err != nil {
		return v1apps.DeploymentSpec{}, err
	}

	volumes := []v1.Volume{
		v1.Volume{
			Name: "name", //TODO change
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: ps.MonitorUserSecretName,
				},
			},
		},
		v1.Volume{
			Name: "proxysql-config",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: ps.NamePrefix + "-proxysql-config",
					},
				},
			},
		},
	}

	return v1apps.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: ps.Labels,
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: ps.Labels,
			},
			Spec: v1.PodSpec{
				Containers:    []v1.Container{proxyContainer},
				NodeSelector:  conf.Instances.Percona.ProxyConfig.NodeSelector,
				RestartPolicy: v1.RestartPolicyAlways,
				Volumes:       volumes,
				Affinity: &v1.Affinity{
					PodAntiAffinity: podAntiAffinity(ps.Labels),
				},
			},
		},
	}, nil
}

func (ps *ProxySQL) initContainer() (v1.Container, error) {
	return v1.Container{
		Name:            "config-generator",
		Image:           "alpine",
		ImagePullPolicy: v1.PullIfNotPresent,
		VolumeMounts: []v1.VolumeMount{
			v1.VolumeMount{
				Name:      "proxysql-config-template",
				MountPath: "/tmp/proxysql.cnf.tmpl",
				SubPath:   "proxysql.cnf.tmpl",
			},
			v1.VolumeMount{
				Name: "shared-data",
				MountPath: "/mnt",
			}
		},
	}, nil
}

func (ps *ProxySQL) proxyContainer() (v1.Container, error) {
	RunAsUser := int64(2)
	AllowPrivilegeEscalation := false

	return v1.Container{
		Name:  "proxysql",
		Image: conf.Instances.Percona.ProxyConfig.Image,
		SecurityContext: &v1.SecurityContext{
			RunAsUser:                &RunAsUser,
			AllowPrivilegeEscalation: &AllowPrivilegeEscalation,
		},
		ImagePullPolicy: v1.PullIfNotPresent,
		Ports: []v1.ContainerPort{
			v1.ContainerPort{
				Name:          "sqlport",
				ContainerPort: ps.Port,
				Protocol:      v1.ProtocolTCP,
			},
		},
		VolumeMounts: []v1.VolumeMount{
			v1.VolumeMount{
				Name:      "proxysql-config",
				MountPath: "/etc/proxysql.cnf", // TODO path
			},
		},
	}, nil
}

func (ps *ProxySQL) buildConfigMap() (v1.ConfigMap, error) {
	configTmpl := conf.Instances.Percona.ProxyConfig.ConfigTemplate + "\n"
	t := template.Must(template.New("config").Parse(configTmpl))

	type backendInfo struct {
		Host, Port, MaxConn string
	}

	var infos []backendInfo
	for _, s := range ps.Servers {
		info := backendInfo{s, strconv.FormatInt(int64(ps.Port), 10), strconv.FormatInt(int64(ps.MaxConn), 10)}
		infos = append(infos, info)
	}

	var outputBuf bytes.Buffer
	if err := t.Execute(&outputBuf, infos); err != nil {
		log.Fatal(err)
	}

	data := map[string]string{
		"proxysql.cnf.tmpl": outputBuf.String(),
	}

	return v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ps.Namespace,
			Name:      ps.NamePrefix + "-proxysql-config-template",
			Labels:    ps.Labels,
		},
		Data: data,
	}, nil
}


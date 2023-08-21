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

package templates

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/db-operator/db-operator/api/v1beta1"
	"github.com/db-operator/db-operator/pkg/consts"
	"github.com/db-operator/db-operator/pkg/utils/database"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/strings/slices"
)

const (
	DEFAULT_TEMPLATE        = "{{ .Protocol }}://{{ .Username }}:{{ .Password }}@{{ .Hostname }}:{{ .Port }}/{{ .Database }}"
	DEFAULT_NAME            = "CONNECTION_STRING"
	TEMPLATE_ANNOTATION_KEY = "kinda.rocks/db-operator-templated-keys"
)

func (tds *TemplateDataSources) BuildVars(templates v1beta1.Templates) error {
	if len(templates) == 0 {
		templates = []*v1beta1.Template{
			{
				Name:     DEFAULT_NAME,
				Template: DEFAULT_TEMPLATE,
				Secret:   true,
			},
		}
	}
	var currentTemplatesSec []string
	var currentTemplatesCm []string
	var result = map[string][]byte{}

	// Get the last applied data
	var lastAppliedSecret []string
	val, ok := tds.SecretK8sObj.ObjectMeta.Annotations[TEMPLATE_ANNOTATION_KEY]
	if ok {
		lastAppliedSecret = strings.Split(val, ",")
	}
	var lastAppliedConfigMap []string
	val, ok = tds.ConfigMapK8sObj.ObjectMeta.Annotations[TEMPLATE_ANNOTATION_KEY]
	if ok {
		lastAppliedConfigMap = strings.Split(val, ",")
	}

	// Populate the blocked data
	// It's requred to get keys that were not added by templates
	var blockedSecretData []string
	for key := range tds.SecretK8sObj.Data {
		if !slices.Contains(lastAppliedSecret, key) {
			blockedSecretData = append(blockedSecretData, key)
		}
	}
	var blockedConfigmapData []string
	for key := range tds.ConfigMapK8sObj.Data {
		if !slices.Contains(lastAppliedConfigMap, key) {
			blockedConfigmapData = append(blockedConfigmapData, key)
		}
	}

	for _, tmpl := range templates {
		t, err := template.New(tmpl.Name).Parse(tmpl.Template)
		if err != nil {
			return err
		}

		var tmplRes bytes.Buffer
		err = t.Execute(&tmplRes, tds)
		if err != nil {
			return err
		}
		result[tmpl.Name] = tmplRes.Bytes()
		if tmpl.Secret {
			if !slices.Contains(blockedSecretData, tmpl.Name) {
				currentTemplatesSec = append(currentTemplatesSec, tmpl.Name)
				tds.SecretK8sObj.Data[tmpl.Name] = tmplRes.Bytes()
			} else {
				return fmt.Errorf("%s already exists in the secret", tmpl.Name)
			}
		} else {
			if !slices.Contains(blockedConfigmapData, tmpl.Name) {
				currentTemplatesCm = append(currentTemplatesCm, tmpl.Name)
				tds.ConfigMapK8sObj.Data[tmpl.Name] = tmplRes.String()
			} else {
				return fmt.Errorf("%s already exists in the configmap", tmpl.Name)
			}

		}
		for _, entry := range lastAppliedSecret {
			if !slices.Contains(currentTemplatesSec, entry) {
				delete(tds.SecretK8sObj.Data, entry)
			}
		}
		for _, entry := range lastAppliedConfigMap {
			if !slices.Contains(currentTemplatesCm, entry) {
				delete(tds.ConfigMapK8sObj.Data, entry)
			}
		}
		tds.SecretK8sObj.ObjectMeta.Annotations[TEMPLATE_ANNOTATION_KEY] = strings.Join(currentTemplatesSec, ",")
		tds.ConfigMapK8sObj.ObjectMeta.Annotations[TEMPLATE_ANNOTATION_KEY] = strings.Join(currentTemplatesCm, ",")

	}
	return nil
}

// TemplateDataSource  should be only the database resource
type TemplateDataSources struct {
	DatabaseK8sObj  *v1beta1.Database
	DbUserK8sObj    *v1beta1.DbUser
	SecretK8sObj    *corev1.Secret
	ConfigMapK8sObj *corev1.ConfigMap
	DatabaseObj     database.Database
	DatabaseUser    *database.DatabaseUser
}

// NewTemplateDataSource is used to init the struct that should handle the templating of secrets and other key-values
// that can be later used by applications.
// If DbUser (second argument) is provided, the templater will be working with a secret that belongs to a dbuser
func NewTemplateDataSource(databaseK8s *v1beta1.Database,
	dbuserK8s *v1beta1.DbUser,
	secretK8s *corev1.Secret,
	configmapK8s *corev1.ConfigMap,
	db database.Database,
	databaseUser *database.DatabaseUser,
) (*TemplateDataSources, error) {
	if databaseK8s == nil {
		return nil, errors.New("database must be passed")
	}
	if secretK8s == nil {
		return nil, errors.New("secret must be passed")
	}
	if configmapK8s == nil {
		return nil, errors.New("configmap must be passed")
	}
	var secretName string
	if dbuserK8s != nil {
		secretName = dbuserK8s.Spec.SecretName
	} else {
		secretName = databaseK8s.Spec.SecretName
	}

	if secretK8s.Name != secretName {
		return nil, fmt.Errorf("secret %s doesn't belong to the database %s", secretK8s.Name, databaseK8s.Name)
	}

	if configmapK8s.Name != databaseK8s.Spec.SecretName {
		return nil, fmt.Errorf("configmap %s doesn't belong to the database %s", secretK8s.Name, databaseK8s.Name)
	}

	return &TemplateDataSources{
		DatabaseK8sObj:  databaseK8s,
		DbUserK8sObj:    dbuserK8s,
		SecretK8sObj:    secretK8s,
		ConfigMapK8sObj: configmapK8s,
		DatabaseObj:     db,
		DatabaseUser:    databaseUser,
	}, nil
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Main getters funcions should be used to query the data
 *  from main data source objects:
 *  - Secret
 *  - Configmap
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// Get the data from the Database Secret
func (tds *TemplateDataSources) Secret(entry string) (string, error) {
	if secret, ok := tds.SecretK8sObj.Data[entry]; ok {
		return string(secret), nil
	}
	return "", fmt.Errorf("entry not found in the secret: %s", entry)
}

// Get the data from the Database ConfigMap
func (tds *TemplateDataSources) ConfigMap(entry string) (string, error) {
	if configmap, ok := tds.ConfigMapK8sObj.Data[entry]; ok {
		return string(configmap), nil
	}
	return "", fmt.Errorf("entry not found in the configmap: %s", entry)
}

// Get the data directly from the database
// TODO: I'm not sure yet how it should work
func (tds *TemplateDataSources) Query(query string) (string, error) {
	result, err := tds.DatabaseObj.QueryAsUser(query, tds.DatabaseUser)
	if err != nil {
		return "", err
	}
	return result, nil
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Helpers should make it easier to access most common values
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

func (tds *TemplateDataSources) Protocol() (string, error) {
	return tds.DatabaseK8sObj.GetProtocol()
}

// Username return the main user username, if dbuser is nil,
// otherwise it returns a name of a DbUser
func (tds *TemplateDataSources) Username() (string, error) {
	engine, err := tds.DatabaseK8sObj.GetEngineType()
	if err != nil {
		return "", err
	}

	switch engine {
	case "postgres":
		return tds.Secret(consts.POSTGRES_USER)
	case "mysql":
		return tds.Secret(consts.MYSQL_USER)
	default:
		return "", fmt.Errorf("unknown engine: %s", engine)
	}
}

// Password return the main user password, if dbuser is nil,
// otherwise it returns a password of a DbUser
func (tds *TemplateDataSources) Password() (string, error) {
	engine, err := tds.DatabaseK8sObj.GetEngineType()
	if err != nil {
		return "", err
	}

	switch engine {
	case "postgres":
		return tds.Secret(consts.POSTGRES_PASSWORD)
	case "mysql":
		return tds.Secret(consts.MYSQL_PASSWORD)
	default:
		return "", fmt.Errorf("unknown engine: %s", engine)
	}
}

func (tds *TemplateDataSources) Database() (string, error) {
	engine, err := tds.DatabaseK8sObj.GetEngineType()
	if err != nil {
		return "", err
	}

	switch engine {
	case "postgres":
		return tds.Secret(consts.POSTGRES_DB)
	case "mysql":
		return tds.Secret(consts.MYSQL_DB)
	default:
		return "", fmt.Errorf("unknown engine: %s", engine)
	}

}

// Hostname
func (tds *TemplateDataSources) Hostname() (string, error) {
	if !tds.DatabaseK8sObj.Status.ProxyStatus.Status {
		dbAddress := tds.DatabaseObj.GetDatabaseAddress()
		return dbAddress.Host, nil
	} else {
		return tds.DatabaseK8sObj.Status.ProxyStatus.ServiceName, nil
	}
}

// Port
func (tds *TemplateDataSources) Port() (int32, error) {
	if !tds.DatabaseK8sObj.Status.ProxyStatus.Status {
		dbAddress := tds.DatabaseObj.GetDatabaseAddress()
		return int32(dbAddress.Port), nil
	} else {
		return tds.DatabaseK8sObj.Status.ProxyStatus.SQLPort, nil
	}
}

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

package templates

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/db-operator/db-operator/v2/api/v1beta1"
	"github.com/db-operator/db-operator/v2/pkg/consts"
	"github.com/db-operator/db-operator/v2/pkg/types"
	"github.com/db-operator/db-operator/v2/pkg/utils/database"
	corev1 "k8s.io/api/core/v1"
)

func (tds *TemplateDataSources) Render(templates v1beta1.Templates) error {
	var currentTemplatesSec []string
	var currentTemplatesCm []string
	result := map[string][]byte{}

	// Get the last applied data
	lastAppliedSecret := getPreviouslyApplied(tds.SecretK8sObj.GetAnnotations())
	lastAppliedConfigMap := getPreviouslyApplied(tds.ConfigMapK8sObj.GetAnnotations())

	// Populate the blocked data
	// It's required to get keys that were not added by templates
	blockedSecretData := getBlockedData(tds.SecretK8sObj.Data, lastAppliedSecret)
	blockedConfigMapData := getBlockedData(tds.ConfigMapK8sObj.Data, lastAppliedConfigMap)

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
			if !isBlocked(blockedSecretData, tmpl.Name) {
				currentTemplatesSec = append(currentTemplatesSec, tmpl.Name)
				tds.SecretK8sObj.Data[tmpl.Name] = tmplRes.Bytes()
			} else {
				return fmt.Errorf("%s already exists in the secret", tmpl.Name)
			}
		} else {
			if !isBlocked(blockedConfigMapData, tmpl.Name) {
				currentTemplatesCm = append(currentTemplatesCm, tmpl.Name)
				tds.ConfigMapK8sObj.Data[tmpl.Name] = tmplRes.String()
			} else {
				return fmt.Errorf("%s already exists in the configmap", tmpl.Name)
			}
		}

	}
	cleanUpData(tds.SecretK8sObj.Data, lastAppliedSecret, currentTemplatesSec)
	cleanUpData(tds.ConfigMapK8sObj.Data, lastAppliedConfigMap, currentTemplatesCm)

	tds.SecretK8sObj.Annotations[consts.TEMPLATE_ANNOTATION_KEY] = strings.Join(currentTemplatesSec, ",")
	tds.ConfigMapK8sObj.Annotations[consts.TEMPLATE_ANNOTATION_KEY] = strings.Join(currentTemplatesCm, ",")

	if len(tds.SecretK8sObj.Annotations[consts.TEMPLATE_ANNOTATION_KEY]) == 0 {
		delete(tds.SecretK8sObj.Annotations, consts.TEMPLATE_ANNOTATION_KEY)
	}

	if len(tds.ConfigMapK8sObj.Annotations[consts.TEMPLATE_ANNOTATION_KEY]) == 0 {
		delete(tds.ConfigMapK8sObj.Annotations, consts.TEMPLATE_ANNOTATION_KEY)
	}

	return nil
}

func getPreviouslyApplied(annotations map[string]string) []string {
	result := []string{}
	val, ok := annotations[consts.TEMPLATE_ANNOTATION_KEY]
	if ok {
		result = strings.Split(val, ",")
	}
	return result
}

func getBlockedData[T string | []byte](data map[string]T, previouslyApplied []string) []string {
	var result []string
	for key := range data {
		if !slices.Contains(previouslyApplied, key) {
			result = append(result, key)
		}
	}
	return result
}

func cleanUpData[T string | []byte](data map[string]T, previouslyApplied, currentlyApplied []string) {
	for _, entry := range previouslyApplied {
		if !slices.Contains(currentlyApplied, entry) {
			delete(data, entry)
		}
	}
}

func isBlocked(blockedKeys []string, key string) bool {
	return slices.Contains(blockedKeys, key)
}

// TemplateDataSource are used by the template renderer to fill the templated values
type TemplateDataSources struct {
	DatabaseK8sObj  *v1beta1.Database
	DbUserK8sObj    *v1beta1.DbUser
	SecretK8sObj    *corev1.Secret
	ConfigMapK8sObj *corev1.ConfigMap
	DatabaseObj     database.Database
	DatabaseUser    *database.DatabaseUser
	// A map of extra values that can be used for templating
	// When used by controllers, these values should be passed
	// from the dbinstance spec
	ExtraTemplateVars map[string]string
}

// NewTemplateDataSource is used to init the struct that should handle the templating of secrets and other key-values
// that can be later used by applications.
// If DbUser (second argument) is provided, the templater will be working with a secret that belongs to a dbuser
func NewTemplateDataSource(
	databaseK8s *v1beta1.Database,
	dbuserk8s *v1beta1.DbUser,
	secretK8s *corev1.Secret,
	configmapK8s *corev1.ConfigMap,
	db database.Database,
	databaseUser *database.DatabaseUser,
	extraTemplateVars map[string]string,
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
	if extraTemplateVars == nil {
		extraTemplateVars = map[string]string{}
	}

	var secretName string
	var caller types.KindaObject
	if dbuserk8s != nil {
		caller = dbuserk8s
		secretName = caller.GetSecretName()
	} else {
		caller = databaseK8s
		secretName = caller.GetSecretName()
	}

	if secretK8s.Name != secretName {
		return nil, fmt.Errorf("secret %s doesn't belong to the %s %s", secretK8s.Name, caller.GetObjectKind().GroupVersionKind().GroupKind().Kind, caller.GetName())
	}

	if configmapK8s.Name != databaseK8s.Spec.SecretName {
		return nil, fmt.Errorf("configmap %s doesn't belong to the database %s", secretK8s.Name, databaseK8s.Name)
	}

	if configmapK8s.Annotations == nil {
		configmapK8s.Annotations = make(map[string]string)
	}
	if secretK8s.Annotations == nil {
		secretK8s.Annotations = make(map[string]string)
	}

	return &TemplateDataSources{
		DatabaseK8sObj:    databaseK8s,
		DbUserK8sObj:      dbuserk8s,
		SecretK8sObj:      secretK8s,
		ConfigMapK8sObj:   configmapK8s,
		DatabaseObj:       db,
		DatabaseUser:      databaseUser,
		ExtraTemplateVars: extraTemplateVars,
	}, nil
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Main getters functions should be used to query the data
 *  from main data source objects:
 *  - Secret
 *  - ConfigMap
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

// Get the data from the extra template vars
func (tds *TemplateDataSources) InstanceVar(name string) (string, error) {
	val, ok := tds.ExtraTemplateVars[name]
	if !ok {
		return "", errors.New("variable is not found")
	}
	return val, nil
}

// Get the data directly from the database
func (tds *TemplateDataSources) Query(query string) (string, error) {
	result, err := tds.DatabaseObj.QueryAsUser(context.TODO(), query, tds.DatabaseUser)
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
	switch tds.DatabaseK8sObj.Status.Engine {
	case "postgres":
		return tds.Secret(consts.POSTGRES_USER)
	case "mysql":
		return tds.Secret(consts.MYSQL_USER)
	default:
		return "", fmt.Errorf("unknown engine: %s", tds.DatabaseK8sObj.Status.Engine)
	}
}

// Password return the main user password, if dbuser is nil,
// otherwise it returns a password of a DbUser
func (tds *TemplateDataSources) Password() (string, error) {
	switch tds.DatabaseK8sObj.Status.Engine {
	case "postgres":
		return tds.Secret(consts.POSTGRES_PASSWORD)
	case "mysql":
		return tds.Secret(consts.MYSQL_PASSWORD)
	default:
		return "", fmt.Errorf("unknown engine: %s", tds.DatabaseK8sObj.Status.Engine)
	}
}

func (tds *TemplateDataSources) Database() (string, error) {
	switch tds.DatabaseK8sObj.Status.Engine {
	case "postgres":
		return tds.Secret(consts.POSTGRES_DB)
	case "mysql":
		return tds.Secret(consts.MYSQL_DB)
	default:
		return "", fmt.Errorf("unknown engine: %s", tds.DatabaseK8sObj.Status.Engine)
	}
}

// Hostname
func (tds *TemplateDataSources) Hostname() (string, error) {
	if !tds.DatabaseK8sObj.Status.ProxyStatus.Status {
		dbAddress := tds.DatabaseObj.GetDatabaseAddress(context.TODO())
		return dbAddress.Host, nil
	} else {
		return tds.DatabaseK8sObj.Status.ProxyStatus.ServiceName, nil
	}
}

// Port
func (tds *TemplateDataSources) Port() (int32, error) {
	if !tds.DatabaseK8sObj.Status.ProxyStatus.Status {
		dbAddress := tds.DatabaseObj.GetDatabaseAddress(context.TODO())
		return int32(dbAddress.Port), nil
	} else {
		return tds.DatabaseK8sObj.Status.ProxyStatus.SQLPort, nil
	}
}

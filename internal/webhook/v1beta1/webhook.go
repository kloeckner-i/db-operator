/*
 * Copyright 2021 kloeckner.i GmbH
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
	"errors"
	"fmt"
	"regexp"
	"slices"

	kindarocksv1beta1 "github.com/db-operator/db-operator/v2/api/v1beta1"
)

var (
	helpers          []string = []string{"Protocol", "Hostname", "Port", "Password", "Username", "Password", "Database"}
	allowedFunctions []string = []string{"Secret", "ConfigMap", "Query", "InstanceVar"}
)

// Make sure that credentials.templates are correct
// ConfigMaps templating is not allowed for DbUser, that's why we have
//
// the second argument: cmAllowed. It should be set to false when
// validation is called by dbuser_webhook.
func ValidateTemplates(templates kindarocksv1beta1.Templates, cmAllowed bool) error {
	for _, template := range templates {
		if !cmAllowed {
			if !template.Secret {
				err := errors.New("ConfigMap templating is not allowed for that kind. Please set .secret to true")
				return err
			}
		}
		// This regexp is getting fields from mustache templates so then they can be compared to allowed fields
		reg := "{{\\s*\\.([\\w\\.]+)\\s*(.*?)\\s*}}"
		r, _ := regexp.Compile(reg)

		fields := r.FindAllStringSubmatch(template.Template, -1)
		for _, field := range fields {
			if validHelperField(field[1]) {
				continue
			}
			if !validFunctionField(field[1]) {
				err := fmt.Errorf("%s is invalid: %v is a field that is not allowed for templating, please use one of these: %v, %v",
					template.Name, field[1], helpers, allowedFunctions)
				return err
			}
			if !validFunctionArg(field[2]) {
				err := fmt.Errorf("%s is invalid: Functions arguments must be not empty and wrapped in quotes, example: {{ .Secret \\\"PASSWORD\\\" }}", template.Name)
				return err
			}
		}
	}
	return nil
}

func ValidateSecretTemplates(templates map[string]string) error {
	for _, template := range templates {
		allowedFields := []string{".Protocol", ".DatabaseHost", ".DatabasePort", ".UserName", ".Password", ".DatabaseName"}
		// This regexp is getting fields from mustache templates so then they can be compared to allowed fields
		reg := "{{\\s*([\\w\\.]+)\\s*}}"
		r, _ := regexp.Compile(reg)
		fields := r.FindAllStringSubmatch(template, -1)
		for _, field := range fields {
			if !slices.Contains(allowedFields, field[1]) {
				err := fmt.Errorf("%v is a field that is not allowed for templating, please use one of these: %v", field[1], allowedFields)
				return err
			}
		}
	}
	return nil
}

func validHelperField(field string) bool {
	return slices.Contains(helpers, field)
}

func validFunctionField(field string) bool {
	return slices.Contains(allowedFunctions, field)
}

func validFunctionArg(field string) bool {
	if len(field) == 0 {
		return false
	}
	functionReg := "\".*\""
	fr, _ := regexp.Compile(functionReg)
	return fr.MatchString(field)
}

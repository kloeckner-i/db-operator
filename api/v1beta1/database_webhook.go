/*
 * Copyright 2021 kloeckner.i GmbH
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
	"errors"
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var databaselog = logf.Log.WithName("database-resource")

func (r *Database) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-kinda-rocks-v1beta1-database,mutating=true,failurePolicy=fail,sideEffects=None,groups=kinda.rocks,resources=databases,verbs=create;update,versions=v1beta1,name=mdatabase.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Database{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Database) Default() {
	databaselog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

//+kubebuilder:webhook:path=/validate-kinda-rocks-v1beta1-database,mutating=false,failurePolicy=fail,sideEffects=None,groups=kinda.rocks,resources=databases,verbs=create;update,versions=v1beta1,name=vdatabase.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Database{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Database) ValidateCreate() error {
	databaselog.Info("validate create", "name", r.Name)

	if r.Spec.SecretsTemplates != nil && r.Spec.Templates != nil {
		return errors.New("using both: secretsTemplates and templates, is not allowed")
	}

	if r.Spec.SecretsTemplates != nil {
		fmt.Printf(`secretsTemplates are deprecated, it will be removed in the next API version. Please, consider switching to templates`)
		if err := ValidateSecretTemplates(r.Spec.SecretsTemplates); err != nil {
			return err
		}

	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Database) ValidateUpdate(old runtime.Object) error {
	databaselog.Info("validate update", "name", r.Name)

	if r.Spec.SecretsTemplates != nil && r.Spec.Templates != nil {
		return errors.New("using both: secretsTemplates and templates, is not allowed")
	}

	if r.Spec.SecretsTemplates != nil {
		fmt.Printf(`secretsTemplates are deprecated, it will be removed in the next API version. Please, consider switching to templates`)
		err := ValidateSecretTemplates(r.Spec.SecretsTemplates)
		if err != nil {
			return err
		}
	}

	// Ensure fields are immutable
	immutableErr := "cannot change %s, the field is immutable"
	oldDatabase, _ := old.(*Database)
	if r.Spec.Instance != oldDatabase.Spec.Instance {
		return fmt.Errorf(immutableErr, "spec.instance")
	}

	if r.Spec.Postgres.Template != oldDatabase.Spec.Postgres.Template {
		return fmt.Errorf(immutableErr, "spec.postgres.template")
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

func ValidateTemplates(templates Templates) error {
	for _, template := range templates {
		fmt.Printf("%s - %s\n", template.Name, template.Template)
		helpers := []string{"Protocol", "Host", "Port", "Password", "Username", "Password", "Database"}
		functions := []string{"Secret", "ConfigMap", "Query"}
		// This regexp is getting fields from mustache templates so then they can be compared to allowed fields
		reg := "{{\\s*\\.([\\w\\.]+)\\s*(.*?)\\s*}}"
		fmt.Println(reg)
		r, _ := regexp.Compile(reg)
		fields := r.FindAllStringSubmatch(template.Template, -1)
		for _, field := range fields {
			fmt.Printf("1-%s\n2-%s\n", field[1], field[2])
			if slices.Contains(helpers, field[1]){
				continue
			} else if slices.Contains(functions, field[1]){
				functionReg := "\".*\""
				fr, _ := regexp.Compile(functionReg)
				if !fr.MatchString(field[2]) {
					err := fmt.Errorf("%s is invalid: Functions must be wrapped in quotes, example: {{ .Secret \\\"PASSWORD\\\" }}", template.Name)
					return err
				}
		  } else {
				err := fmt.Errorf("%s is invalid: %v is a field that is not allowed for templating, please use one of these: %v, %v", 
				template.Name, field[1], helpers, functions)
				return err
			}
		}
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Database) ValidateDelete() error {
	databaselog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

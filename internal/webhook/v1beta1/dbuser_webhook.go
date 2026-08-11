/*
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
	"context"
	"errors"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kindarocksv1beta1 "github.com/db-operator/db-operator/v2/api/v1beta1"
	"github.com/db-operator/db-operator/v2/pkg/consts"
)

// nolint:unused
// log is for logging in this package.
var dbuserlog = logf.Log.WithName("dbuser-resource")

// SetupDbUserWebhookWithManager registers the webhook for DbUser in the manager.
func SetupDbUserWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &kindarocksv1beta1.DbUser{}).
		WithValidator(&DbUserCustomValidator{}).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-kinda-rocks-v1beta1-dbuser,mutating=false,failurePolicy=fail,sideEffects=None,groups=kinda.rocks,resources=dbusers,verbs=create;update,versions=v1beta1,name=vdbuser-v1beta1.kb.io,admissionReviewVersions=v1

// DbUserCustomValidator struct is responsible for validating the DbUser resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type DbUserCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type DbUser.
func (v *DbUserCustomValidator) ValidateCreate(_ context.Context, obj *kindarocksv1beta1.DbUser) (admission.Warnings, error) {
	dbuserlog.Info("Validation for DbUser upon creation", "name", obj.GetName())

	warnings := []string{}
	if len(obj.Spec.ExtraPrivileges) > 0 {
		warnings = append(warnings,
			"extra privileges is an experimental feature, please use at your own risk and feel free to open GitHub issues.")
	}

	if err := TestExtraPrivileges(obj.Spec.ExtraPrivileges); err != nil {
		return warnings, err
	}
	if err := kindarocksv1beta1.IsAccessTypeSupported(obj.Spec.AccessType); err != nil {
		return warnings, err
	}

	return warnings, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type DbUser.
func (v *DbUserCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *kindarocksv1beta1.DbUser) (admission.Warnings, error) {
	dbuserlog.Info("Validation for DbUser upon update", "name", newObj.GetName())

	warnings := []string{}
	if len(newObj.Spec.ExtraPrivileges) > 0 {
		warnings = append(warnings,
			"extra privileges is an experimental feature, please use at your own risk and feel free to open GitHub issues.")
	}
	if err := TestExtraPrivileges(newObj.Spec.ExtraPrivileges); err != nil {
		return warnings, err
	}

	if len(oldObj.Spec.ExistingUser) > 0 && len(newObj.Spec.ExistingUser) == 0 {
		warnings = append(warnings, "After swtching from existing user to a generated user, the password is set to an empty string, remove the db secret to generate it")
	}

	if err := kindarocksv1beta1.IsAccessTypeSupported(newObj.Spec.AccessType); err != nil {
		return warnings, err
	}
	if newObj.Spec.Credentials.Templates != nil {
		if err := ValidateTemplates(newObj.Spec.Credentials.Templates, false); err != nil {
			return warnings, err
		}
	}

	return warnings, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type DbUser.
func (v *DbUserCustomValidator) ValidateDelete(ctx context.Context, obj *kindarocksv1beta1.DbUser) (admission.Warnings, error) {
	dbuserlog.Info("Validation for DbUser upon deletion", "name", obj.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}

func TestExtraPrivileges(privileges []string) error {
	for _, privilege := range privileges {
		if strings.ToUpper(privilege) == consts.ALL_PRIVILEGES {
			return errors.New("it's not allowed to grant ALL PRIVILEGES")
		}
	}
	return nil
}

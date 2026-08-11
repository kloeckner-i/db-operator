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
	"fmt"
	"slices"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kindarocksv1beta1 "github.com/db-operator/db-operator/v2/api/v1beta1"
	"github.com/db-operator/db-operator/v2/internal/helpers/kube"
	"github.com/db-operator/db-operator/v2/pkg/consts"
)

// nolint:unused
// log is for logging in this package.
var dbinstancelog = logf.Log.WithName("dbinstance-resource")

// SetupDbInstanceWebhookWithManager registers the webhook for DbInstance in the manager.
func SetupDbInstanceWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &kindarocksv1beta1.DbInstance{}).
		WithValidator(&DbInstanceCustomValidator{}).
		WithDefaulter(&DbInstanceCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-kinda-rocks-v1beta1-dbinstance,mutating=true,failurePolicy=fail,sideEffects=None,groups=kinda.rocks,resources=dbinstances,verbs=create;update,versions=v1beta1,name=mdbinstance-v1beta1.kb.io,admissionReviewVersions=v1

// DbInstanceCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind DbInstance when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type DbInstanceCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind DbInstance.
func (d *DbInstanceCustomDefaulter) Default(_ context.Context, obj *kindarocksv1beta1.DbInstance) error {
	dbinstancelog.Info("Defaulting for DbInstance", "name", obj.GetName())
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-kinda-rocks-v1beta1-dbinstance,mutating=false,failurePolicy=fail,sideEffects=None,groups=kinda.rocks,resources=dbinstances,verbs=create;update,versions=v1beta1,name=vdbinstance-v1beta1.kb.io,admissionReviewVersions=v1

// DbInstanceCustomValidator struct is responsible for validating the DbInstance resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type DbInstanceCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

func TestAllowedPrivileges(privileges []string) error {
	for _, privilege := range privileges {
		if strings.ToUpper(privilege) == consts.ALL_PRIVILEGES {
			return errors.New("it's not allowed to grant ALL PRIVILEGES")
		}
	}
	return nil
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type DbInstance.
func (v *DbInstanceCustomValidator) ValidateCreate(_ context.Context, obj *kindarocksv1beta1.DbInstance) (admission.Warnings, error) {
	dbinstancelog.Info("Validation for DbInstance upon creation", "name", obj.GetName())

	if err := TestAllowedPrivileges(obj.Spec.AllowedPrivileges); err != nil {
		return nil, err
	}

	if obj.Spec.Google != nil {
		dbinstancelog.Info("Google instances are deprecated, and will be removed in v1beta2")
	}

	if err := ValidateConfigVsConfigFrom(obj.Spec.Generic); err != nil {
		return nil, err
	}
	if err := ValidateEngine(obj.Spec.Engine); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type DbInstance.
func (v *DbInstanceCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *kindarocksv1beta1.DbInstance) (admission.Warnings, error) {
	dbinstancelog.Info("Validation for DbInstance upon update", "name", newObj.GetName())

	if err := TestAllowedPrivileges(newObj.Spec.AllowedPrivileges); err != nil {
		return nil, err
	}

	if newObj.Spec.Google != nil {
		dbinstancelog.Info("Google instances are deprecated, and will be removed in v1beta2")
	}

	if err := ValidateConfigVsConfigFrom(newObj.Spec.Generic); err != nil {
		return nil, err
	}

	immutableErr := "cannot change %s, the field is immutable"
	if newObj.Spec.Engine != oldObj.Spec.Engine {
		return nil, fmt.Errorf(immutableErr, "engine")
	}
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type DbInstance.
func (v *DbInstanceCustomValidator) ValidateDelete(ctx context.Context, obj *kindarocksv1beta1.DbInstance) (admission.Warnings, error) {
	dbinstancelog.Info("Validation for DbInstance upon deletion", "name", obj.GetName())

	return nil, nil
}

func ValidateConfigVsConfigFrom(r *kindarocksv1beta1.GenericInstance) error {
	if r != nil {
		if len(r.Host) > 0 && r.HostFrom != nil {
			return errors.New("it's not allowed to use both host and hostFrom, please choose one")
		}
		if len(r.PublicIP) > 0 && r.PublicIPFrom != nil {
			return errors.New("it's not allowed to use both publicIp and publicIpFrom, please choose one")
		}
		if r.Port > 0 && r.PortFrom != nil {
			return errors.New("it's not allowed to use both port and portFrom, please choose one")
		}
	}
	return nil
}

func ValidateConfigFrom(dbin *kindarocksv1beta1.GenericInstance) error {
	check := dbin.HostFrom
	if check != nil && check.Kind != kube.CONFIGMAP && check.Kind != kube.SECRET {
		return fmt.Errorf("unsupported kind in hostFrom: %s, please use %s or %s", check.Kind, kube.CONFIGMAP, kube.SECRET)
	}
	check = dbin.PortFrom
	if check != nil && check.Kind != kube.CONFIGMAP && check.Kind != kube.SECRET {
		return fmt.Errorf("unsupported kind in portFrom: %s, please use %s or %s", check.Kind, kube.CONFIGMAP, kube.SECRET)
	}
	check = dbin.PublicIPFrom
	if check != nil && check.Kind != kube.CONFIGMAP && check.Kind != kube.SECRET {
		return fmt.Errorf("unsupported kind in publicIpFrom: %s, please use %s or %s", check.Kind, kube.CONFIGMAP, kube.SECRET)
	}
	return nil
}

func ValidateEngine(engine string) error {
	if !(slices.Contains([]string{"postgres", "mysql"}, engine)) {
		return fmt.Errorf("unsupported engine: %s. please use either postgres or mysql", engine)
	}
	return nil
}

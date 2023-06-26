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

package v1beta1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var dbuserlog = logf.Log.WithName("dbuser-resource")

func (r *DbUser) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-kinda-rocks-v1beta1-dbuser,mutating=false,failurePolicy=fail,sideEffects=None,groups=kinda.rocks,resources=dbusers,verbs=create;update,versions=v1beta1,name=vdbuser.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &DbUser{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *DbUser) ValidateCreate() error {
	dbuserlog.Info("validate create", "name", r.Name)
	if err := IsAccessTypeSupported(r.Spec.AccessType); err != nil {
		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *DbUser) ValidateUpdate(old runtime.Object) error {
	dbuserlog.Info("validate update", "name", r.Name)
	if err := IsAccessTypeSupported(r.Spec.AccessType); err != nil {
		return err
	}
	_, ok := old.(*DbUser)
	if !ok {
		return fmt.Errorf("couldn't get the previous version of %s", r.Name)
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *DbUser) ValidateDelete() error {
	dbuserlog.Info("validate delete", "name", r.Name)
	return nil
}

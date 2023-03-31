/*
 * Copyright 2021 kloeckner.i GmbH
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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var dbinstancelog = logf.Log.WithName("dbinstance-resource")

func (r *DbInstance) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-kci-rocks-v1beta1-dbinstance,mutating=true,failurePolicy=fail,sideEffects=None,groups=kinda.rocks,resources=dbinstances,verbs=create;update,versions=v1beta1,name=mdbinstance.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &DbInstance{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *DbInstance) Default() {
	dbinstancelog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-kci-rocks-v1beta1-dbinstance,mutating=false,failurePolicy=fail,sideEffects=None,groups=kinda.rocks,resources=dbinstances,verbs=create;update,versions=v1beta1,name=vdbinstance.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &DbInstance{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *DbInstance) ValidateCreate() error {
	dbinstancelog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *DbInstance) ValidateUpdate(old runtime.Object) error {
	dbinstancelog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *DbInstance) ValidateDelete() error {
	dbinstancelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

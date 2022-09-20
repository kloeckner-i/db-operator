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

package controllers

import (
	"strings"

	kciv1alpha1 "github.com/kloeckner-i/db-operator/api/v1alpha1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	DbSecretAnnotation = "db-operator/database"
)

/* ------ Secret Event Handler ------ */
type secretEventHandler struct {
	client.Client
}

func (e *secretEventHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	logrus.Info("Start processing Database Secret Update Event")

	switch v := evt.ObjectNew.(type) {

	default:
		logrus.Error("Database Secret Update Event error! Unknown object: type=", v.GetObjectKind(), ", name=", evt.ObjectNew.GetNamespace(), "/", evt.ObjectNew.GetName())
		return

	case *corev1.Secret:
		// only annotated secrets are watched
		secretNew := evt.ObjectNew.(*corev1.Secret)
		annotations := secretNew.ObjectMeta.GetAnnotations()
		dbSecretAnnotation, ok := annotations[DbSecretAnnotation]
		if !ok {
			logrus.Error("Database Secret Update Event error! Annotation '", DbSecretAnnotation, "' value is empty or not exist.")
			return
		}

		logrus.Info("Processing Database Secret annotation: name=", DbSecretAnnotation, ", value=", dbSecretAnnotation)

		dbcrNames := strings.Split(dbSecretAnnotation, ",")
		for _, dbcrName := range dbcrNames {
			// send Database Reconcile Request
			logrus.Info("Database Secret has been changed and related Database resource will be reconciled: secret=", secretNew.Namespace, "/", secretNew.Name, ", database=", dbcrName)
			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: secretNew.GetNamespace(),
				Name:      dbcrName,
			}})
		}
	}
}

func (e *secretEventHandler) Delete(event.DeleteEvent, workqueue.RateLimitingInterface) {
	logrus.Error("secretEventHandler.Delete(...) event has been FIRED but NOT implemented!")
}

func (e *secretEventHandler) Generic(event.GenericEvent, workqueue.RateLimitingInterface) {
	logrus.Error("secretEventHandler.Generic(...) event has been FIRED but NOT implemented!")
}

func (e *secretEventHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	logrus.Error("secretEventHandler.Create(...) event has been FIRED but NOT implemented!")
}

/* ------ Event Filter Functions ------ */

func isWatchedNamespace(watchNamespaces []string, ro runtime.Object) bool {
	if watchNamespaces[0] == "" { // # it's necessary to set "" to watch cluster wide
		return true // watch for all namespaces
	}
	// define object's namespace
	objectNamespace := ""
	database, isDatabase := ro.(*kciv1alpha1.Database)
	if isDatabase {
		objectNamespace = database.Namespace
	} else {
		secret, isSecret := ro.(*corev1.Secret)
		if isSecret {
			objectNamespace = secret.Namespace
		} else {
			logrus.Info("unknown object", "object", ro)
			return false
		}
	}

	// check that current namespace is watched by db-operator
	for _, ns := range watchNamespaces {
		if ns == objectNamespace {
			return true
		}
	}
	return false
}

func isDatabase(ro runtime.Object) bool {
	_, isDatabase := ro.(*kciv1alpha1.Database)
	return isDatabase
}

func isObjectUpdated(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		logrus.Error(nil, "Update event has no old runtime object to update", "event", e)
		return false
	}
	if e.ObjectNew == nil {
		logrus.Error(nil, "Update event has no new runtime object for update", "event", e)
		return false
	}
	// if object kind is a Database check that 'metadata.generation' field ('spec' section) has been changed
	_, isDatabase := e.ObjectNew.(*kciv1alpha1.Database)
	if isDatabase {
		return e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration()
	}

	// if object kind is a Secret check that password value has changed
	secretNew, isSecret := e.ObjectNew.(*corev1.Secret)
	if isSecret {
		// only annotated secrets are watched
		annotations := secretNew.ObjectMeta.GetAnnotations()
		dbcrName, ok := annotations[DbSecretAnnotation]
		if !ok {
			return false // no annotation found
		}
		logrus.Info("Secret Update Event detected: secret=", secretNew.Namespace, "/", secretNew.Name, ", database=", dbcrName)
		return true
	}
	return false // unknown object, ignore Update Event
}

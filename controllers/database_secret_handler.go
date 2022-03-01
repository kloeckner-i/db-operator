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
	"bytes"
	"context"
	"fmt"
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
	PgPasswordKey    = "POSTGRES_PASSWORD"
	MysqlPasswordKey = "PASSWORD"
	PgEngine         = "postgres"
	MysqlEngine      = "mysql"
)

/* ------ Secret Event Handler ------ */
type secretEventHandler struct {
	client.Client
}

func (e *secretEventHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {

	logrus.Info("Secret Update Event processing started")

	switch v := evt.ObjectNew.(type) {

	default:
		logrus.Error("Secret Update Event error! Unknown object: type=", v.GetObjectKind(), ", name=", evt.ObjectNew.GetNamespace(), "/", evt.ObjectNew.GetName())
		return

	case *corev1.Secret:
		secretNew := evt.ObjectNew.(*corev1.Secret)
		// find database object
		databases, dbNames, _ := getDatabasesBySecret(e.Client, types.NamespacedName{
			Name:      evt.ObjectNew.GetName(),
			Namespace: evt.ObjectNew.GetNamespace(),
		})

		numDatabases := len(databases)
		if numDatabases == 0 {
			logrus.Error("Secret Update Event error! Could not find Database resource related to the Secret: secret=", secretNew.Namespace, "/", secretNew.Name)
			return
		}
		if numDatabases > 1 {
			// We do not allow using the same Secret resource for several Database resources!
			logrus.Warning("Secret Update Event error! Multiple Database resources related to the same Secret: secret=", secretNew.Namespace, "/", secretNew.Name, ", dbNames=", dbNames)
		}

		database := databases[0]

		// make sure that new credentials are valid
		_, err := parseDatabaseSecretData(&database, secretNew.Data)
		if err != nil {
			logrus.Error("Secret Update Event error! New Secret Data contains incorrect credentials: secret=", secretNew.Namespace, "/", secretNew.Name)
			return
		}

		logrus.Info("Database Secret Data has been changed and related Database resource will be reconciled: database=", database.Namespace, "/", database.Name, ", secret=", secretNew.Namespace, "/", secretNew.Name)
		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: database.GetNamespace(),
			Name:      database.GetName(),
		}})
	}

	logrus.Info("Secret Update Event successfully processed")
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

func getDatabasesBySecret(c client.Client, secret types.NamespacedName) ([]kciv1alpha1.Database, []string, error) {
	databaseList := kciv1alpha1.DatabaseList{}
	if err := c.List(context.Background(), &databaseList, &client.ListOptions{Namespace: secret.Namespace}); err != nil {
		return nil, nil, err
	}
	var matchedDbs []kciv1alpha1.Database
	var matchedDbNames []string
	for _, database := range databaseList.Items {
		if database.Spec.SecretName == secret.Name {
			matchedDbs = append(matchedDbs, database)
			matchedDbNames = append(matchedDbNames, database.Name)
		}
	}
	return matchedDbs, matchedDbNames, nil
}

func isFieldUpdated(dataOld map[string][]byte, dataNew map[string][]byte, keyName string) bool {

	// read old and new value
	oldValue, oldValueOk := dataOld[keyName]
	newValue, newValueOk := dataNew[keyName]

	if !oldValueOk || !newValueOk {
		return false // values empty or do not exist
	}

	result := bytes.Compare(oldValue, newValue)
	return result != 0 // true if values are not empty and updated
}

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
		// if object kind is a Secret let's check that DB password inside has been updated
		secretOld, secretOldOk := e.ObjectOld.(*corev1.Secret)
		if !secretOldOk {
			logrus.Error(nil, "Update Event error! ObjectOld is NOT a Secret: e.ObjectOld.GetName=", e.ObjectOld.GetName())
			return false
		}

		engine, passwordKey, err := resolveEngine(secretNew)
		if err != nil {
			return false // cannot resolve DB engine, ignore Update Event
		}

		isPasswordUpdated := isFieldUpdated(secretOld.Data, secretNew.Data, passwordKey)
		logrus.Info("Secret Update Event Detected: engine=", engine, ", secret=", secretNew.Namespace, "/", secretNew.Name, ", isPasswordUpdated=", isPasswordUpdated)
		return isPasswordUpdated
	}
	return false // unknown object, ignore Update Event
}

func resolveEngine(secret *corev1.Secret) (string, string, error) {

	// Resolve engine by presence of corresponding secret key
	if _, valueOk := secret.Data[PgPasswordKey]; valueOk {
		return PgEngine, PgPasswordKey, nil
	}

	if _, valueOk := secret.Data[MysqlPasswordKey]; valueOk {
		return MysqlEngine, MysqlPasswordKey, nil
	}

	return "", "", fmt.Errorf("could not resolve DB engine by password key: secret='%v/%v'", secret.Namespace, secret.Name)
}

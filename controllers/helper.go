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
	"context"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	
	kciv1alpha1 "github.com/kloeckner-i/db-operator/api/v1alpha1"
	"github.com/kloeckner-i/db-operator/pkg/utils/kci"
	crdv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func isDBSpecChanged(dbcr *kciv1alpha1.Database) bool {
	annotations := dbcr.ObjectMeta.GetAnnotations()

	return annotations["checksum/spec"] != kci.GenerateChecksum(dbcr.Spec)
}

func addDBSpecChecksum(dbcr *kciv1alpha1.Database) {
	annotations := dbcr.ObjectMeta.GetAnnotations()
	if len(annotations) == 0 {
		annotations = make(map[string]string)
	}

	annotations["checksum/spec"] = kci.GenerateChecksum(dbcr.Spec)
	dbcr.ObjectMeta.SetAnnotations(annotations)
}

func isDBInstanceSpecChanged(ctx context.Context, dbin *kciv1alpha1.DbInstance) bool {
	checksums := dbin.Status.Checksums
	if checksums["spec"] != kci.GenerateChecksum(dbin.Spec) {
		return true
	}

	if backend, _ := dbin.GetBackendType(); backend == "google" {
		instanceConfig, _ := kci.GetConfigResource(ctx, dbin.Spec.Google.ConfigmapName.ToKubernetesType())
		if checksums["config"] != kci.GenerateChecksum(instanceConfig) {
			return true
		}
	}

	return false
}

func addDBInstanceChecksumStatus(ctx context.Context, dbin *kciv1alpha1.DbInstance) {
	checksums := dbin.Status.Checksums
	if len(checksums) == 0 {
		checksums = make(map[string]string)
	}
	checksums["spec"] = kci.GenerateChecksum(dbin.Spec)

	if backend, _ := dbin.GetBackendType(); backend == "google" {
		instanceConfig, _ := kci.GetConfigResource(ctx, dbin.Spec.Google.ConfigmapName.ToKubernetesType())
		checksums["config"] = kci.GenerateChecksum(instanceConfig)
	}

	dbin.Status.Checksums = checksums
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// inCrdList returns true if monitoring is enabled in DbInstance spec.
func inCrdList(crds crdv1.CustomResourceDefinitionList, api string) bool {
	for _, crd := range crds.Items {
		if crd.Name == api {
			return true
		}
	}
	return false
}


/* ------ Secret Update Event Handler ------ */
type secretEventHandler struct {
	client.Client
}

func (e *secretEventHandler) Delete(event.DeleteEvent, workqueue.RateLimitingInterface) {
	logrus.Error("secretEventHandler.Delete(...) event has been FIRED but NOT implemented!")
	return
}
func (e *secretEventHandler) Generic(event.GenericEvent, workqueue.RateLimitingInterface) {
	logrus.Error("secretEventHandler.Generic(...) event has been FIRED but NOT implemented!")
	return
}
func (e *secretEventHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	logrus.Error("secretEventHandler.Create(...) event has been FIRED but NOT implemented!")
	return
}
func (e *secretEventHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	
	logrus.Info("secretEventHandler.Update event has been started")
	
	switch v := evt.ObjectNew.(type) {
	
	default:
		logrus.Error("secretEventHandler.Update ERROR: unknown object: type=`%s`, name=`%s`", v.GetObjectKind(), evt.ObjectNew.GetName())
		return
	
	case *corev1.Secret:
		secretNew := evt.ObjectNew.(*corev1.Secret)
		
		databases, _ := getDatabasesBySecret(e.Client, types.NamespacedName{
			Name:      evt.ObjectNew.GetName(),
			Namespace: evt.ObjectNew.GetNamespace(),
		})
		
		for _, database := range databases {
			
			// make sure that new credentials are valid
			_, err := parseDatabaseSecretData(&database, secretNew.Data)
			if err != nil {
				logrus.Error("secretEventHandler.Update ERROR: secretNew.Data has incorrect credentials", "secretNew.Name", secretNew.Name)
				return
			}
			
			// make sure that old credentials were valid
			secretOld := evt.ObjectOld.(*corev1.Secret)
			_, err = parseDatabaseSecretData(&database, secretOld.Data)
			if err != nil {
				logrus.Error("secretEventHandler.Update ERROR: secretOld.Data has incorrect credentials", "secretOld.Name", secretOld.Name)
				return
			}
			
			logrus.Info("Database SecretData value has changed and Database resource will be reconciled", "ns", database.GetNamespace(), "secret", evt.ObjectNew.GetName(), "database", database.GetName())
			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: database.GetNamespace(),
				Name:      database.GetName(),
			}})
		}
	}
	
	logrus.Info("secretEventHandler.Update event successfully processed")
}


func getDatabasesBySecret(c client.Client, secret types.NamespacedName) ([]kciv1alpha1.Database, error) {
	databaseList :=kciv1alpha1.DatabaseList{}
	if err := c.List(context.Background(), &databaseList, &client.ListOptions{Namespace: secret.Namespace}); err != nil {
		return nil, err
	}
	var matched []kciv1alpha1.Database
	for _, database := range databaseList.Items {
		if database.Spec.SecretName == secret.Name {
			matched = append(matched, database)
		}
	}
	return matched, nil
}


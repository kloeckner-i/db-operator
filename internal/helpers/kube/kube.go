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

package kube

import (
	"context"
	"fmt"

	"github.com/db-operator/db-operator/v2/pkg/consts"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dbotypes "github.com/db-operator/db-operator/v2/pkg/types"
)

const ERROR_CANT_CAST = "couldn't cast a caller to the client.Object"

/*
All the actions that interact with K8s API in order to edit objects
that do not belong to db-operator (e.g. Secrets, ConfigMaps) should be
edited with this KubeHelper, because it's taking care of setting
required metadata fields, that later will be used by db-operator
to understand whether it can or can not used an external object
for an internal resource
*/
type KubeHelper struct {
	Cli client.Client
	Rec events.EventRecorder
	// Caller is a db-operator object that is requesting modifications
	Caller dbotypes.KindaObject
}

// Init a Kubehelper struct
func NewKubeHelper(cli client.Client, rec events.EventRecorder, caller dbotypes.KindaObject) *KubeHelper {
	return &KubeHelper{cli, rec, caller}
}

// Generic function that will either cleanup or update an object
// depending on the status of the caller
func (kh *KubeHelper) ModifyObject(ctx context.Context, obj client.Object) error {
	// If the caller is removed, the target object should be cleaned up
	if kh.Caller.IsDeleted() {
		return kh.HandleDelete(ctx, obj)
	}

	return kh.HandleCreateOrUpdate(ctx, obj)
}

// If KindaRocks object is removed, other objects should be cleaned up
func (kh *KubeHelper) HandleDelete(ctx context.Context, obj client.Object) error {
	// If object is not used by the caller, we shouldn't edit it when a caller is removed
	if err := kh.Cli.Get(ctx, types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, obj); err != nil {
		return err
	}
	if !kh.IsUsedByCaller(obj) {
		return fmt.Errorf("%s %s is not used by %s %s, editing is not possible",
			obj.GetObjectKind().GroupVersionKind().Kind,
			obj.GetName(),
			kh.Caller.GetObjectKind().GroupVersionKind().Kind,
			kh.Caller.GetName(),
		)
	}
	obj = kh.DeleteUsedByLabels(obj)
	if err := kh.Cli.Update(ctx, obj); err != nil {
		return err
	}
	return nil
}

func (kh *KubeHelper) HandleCreateOrUpdate(ctx context.Context, obj client.Object) error {
	var err error
	newObj := obj.DeepCopyObject().(client.Object)

	if len(obj.GetResourceVersion()) == 0 {
		newObj, err = kh.Create(ctx, obj)
		if err != nil {
			return err
		}
	}

	return kh.Update(ctx, newObj)
}

func (kh *KubeHelper) Create(ctx context.Context, obj client.Object) (client.Object, error) {
	err := kh.Cli.Create(ctx, obj)
	log := log.FromContext(ctx)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		log.Error(err, "couldn't create a Kubernetes resource", "kind", obj.GetObjectKind().GroupVersionKind().Kind, "name", obj.GetName())
		return nil, err
	}
	// Return an updated object already
	// I'm not sure how to make it better
	refreshedObj := obj.DeepCopyObject().(client.Object)
	if err := kh.Cli.Get(ctx, types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, refreshedObj); err != nil {
		log.Error(err, "сouldn't get a Kubernetes resource", "kind", obj.GetObjectKind().GroupVersionKind().Kind, "name", obj.GetName())
		return nil, err
	}
	return refreshedObj, nil
}

func (kh *KubeHelper) Update(ctx context.Context, obj client.Object) error {
	if !kh.IsUsedByAny(obj) {
		obj = kh.SetUsedByLabels(obj)
		// Set the labels right away to minimize the chance of a resource being hijacked
		if err := kh.Cli.Update(ctx, obj); err != nil {
			return err
		}
	}
	// Check if the object isn't used by a caller
	if !kh.IsUsedByCaller(obj) {
		return fmt.Errorf("%s %s is not used by %s %s, editing is not possible",
			obj.GetObjectKind().GroupVersionKind().Kind,
			obj.GetName(),
			kh.Caller.GetObjectKind().GroupVersionKind().Kind,
			kh.Caller.GetName(),
		)
	}
	// Set owner reference
	ownerRef := kh.BuildOwnerReference()
	obj = kh.SetOwnerReference(obj, ownerRef)
	return kh.Cli.Update(ctx, obj)
}

func (kh *KubeHelper) IsUsedByAny(obj client.Object) bool {
	labels := obj.GetLabels()
	// It's enough to check just kind label to see if an object is used by db-operator
	_, ok := labels[consts.USED_BY_KIND_LABEL_KEY]
	return ok
}

func (kh *KubeHelper) IsUsedByCaller(obj client.Object) bool {
	wishedLabels := kh.BuildUsedByLabels()
	labels := obj.GetLabels()
	res := labels[consts.USED_BY_KIND_LABEL_KEY] == wishedLabels[consts.USED_BY_KIND_LABEL_KEY] &&
		labels[consts.USED_BY_NAME_LABEL_KEY] == wishedLabels[consts.USED_BY_NAME_LABEL_KEY]
	return res
}

func (kh *KubeHelper) SetOwnerReference(obj client.Object, ownerRef metav1.OwnerReference) client.Object {
	// It's required in order to remove the owner reference
	// that was previously set by the cleanup feature
	newOwnerRefs := []metav1.OwnerReference{}
	for _, ref := range obj.GetOwnerReferences() {
		if ref.UID != kh.Caller.GetUID() {
			newOwnerRefs = append(newOwnerRefs, ref)
		}
	}

	if ownerRef != (metav1.OwnerReference{}) {
		newOwnerRefs = append(newOwnerRefs, ownerRef)
	}

	obj.SetOwnerReferences(newOwnerRefs)
	return obj
}

func (kh *KubeHelper) BuildUsedByLabels() map[string]string {
	labels := map[string]string{
		consts.USED_BY_KIND_LABEL_KEY: kh.Caller.GetObjectKind().GroupVersionKind().Kind,
		consts.USED_BY_NAME_LABEL_KEY: kh.Caller.GetName(),
	}
	return labels
}

func (kh *KubeHelper) SetUsedByLabels(objTarget client.Object) client.Object {
	existingLabels := objTarget.GetLabels()
	newLabels := kh.BuildUsedByLabels()
	maps.Copy(newLabels, existingLabels)
	objTarget.SetLabels(newLabels)
	return objTarget
}

func (kh *KubeHelper) DeleteUsedByLabels(obj client.Object) client.Object {
	labels := obj.GetLabels()
	delete(labels, consts.USED_BY_KIND_LABEL_KEY)
	delete(labels, consts.USED_BY_NAME_LABEL_KEY)
	return obj
}

func (kh *KubeHelper) BuildOwnerReference() metav1.OwnerReference {
	ownership := metav1.OwnerReference{}
	// If cleanup is true, all the objects that are created by db-operator should be owned by the database object,
	//  so they are removed when database is removed
	if kh.Caller.IsCleanup() {
		APIVersion := fmt.Sprintf("%s/%s", kh.Caller.GetObjectKind().GroupVersionKind().Group, kh.Caller.GetObjectKind().GroupVersionKind().Version)
		ownership = metav1.OwnerReference{
			APIVersion: APIVersion,
			Kind:       kh.Caller.GetObjectKind().GroupVersionKind().Kind,
			Name:       kh.Caller.GetName(),
			UID:        kh.Caller.GetUID(),
		}
	}
	return ownership
}

const (
	SECRET    = "Secret"
	CONFIGMAP = "ConfigMap"
)

func (kh *KubeHelper) GetValueFrom(ctx context.Context, kind, namespace, name, key string) (string, error) {
	nsName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	switch kind {
	case SECRET:
		obj := &corev1.Secret{}
		if err := kh.Cli.Get(ctx, nsName, obj); err != nil {
			return "", err
		}
		val, ok := GetValueByKey(obj.Data, key)
		if !ok {
			return "", fmt.Errorf("secret %s/%s doesn't contain key %s", namespace, name, key)
		}
		return string(val), nil
	case CONFIGMAP:
		obj := &corev1.ConfigMap{}
		if err := kh.Cli.Get(ctx, nsName, obj); err != nil {
			return "", err
		}
		val, ok := GetValueByKey(obj.Data, key)
		if !ok {
			return "", fmt.Errorf("secret %s/%s doesn't contain key %s", namespace, name, key)
		}
		return val, nil
	default:
		err := fmt.Errorf("unknown source kind: %s, please use %s or %s", kind, CONFIGMAP, SECRET)
		return "", err
	}
}

func GetValueByKey[T string | []byte](data map[string]T, key string) (T, bool) {
	val, ok := data[key]
	return val, ok
}

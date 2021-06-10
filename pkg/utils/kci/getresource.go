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

package kci

import (
	"context"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// GetConfigResource get configmap resource by kubernetes incluster rest api
// TODO: will be deprecated
func GetConfigResource(ctx context.Context, key types.NamespacedName) (*corev1.ConfigMap, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	cm, err := clientset.CoreV1().ConfigMaps(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return cm, err
}

// GetSecretResource get secret resource by kubernetes incluster rest api
// TODO: will be deprecated
func GetSecretResource(ctx context.Context, key types.NamespacedName) (*corev1.Secret, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	secret, err := clientset.CoreV1().Secrets(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		logrus.Errorf("secret %s not found", key.Name)
		return nil, err
	} else if statusError, isStatus := err.(*k8serrors.StatusError); isStatus {
		logrus.Errorf("error getting Secret %s %v\n", key.Name, statusError.ErrStatus.Message)
		return nil, err
	} else if err != nil {
		return nil, err
	} else {
		return secret, nil
	}
}

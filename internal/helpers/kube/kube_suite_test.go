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

package kube_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	kindav1beta1 "github.com/db-operator/db-operator/v2/api/v1beta1"
	"github.com/db-operator/db-operator/v2/internal/helpers/kube"
	"github.com/db-operator/db-operator/v2/pkg/consts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	"k8s.io/kubectl/pkg/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	cfg       *rest.Config
	k8sClient client.Client // You'll be using this client in your tests.
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
)

func TestKubernetesHelper(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "KubeHelpers test")
}

// Prepare the test environments
var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.TODO())
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: filepath.Join("..", "..", "..", "bin", "k8s",
			fmt.Sprintf("1.36.2-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}
	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	err = kindav1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})

	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = Describe("KubeHelpers test", func() {
	Context("Test NewKubeHelper function", func() {
		It("Simply test that kubehelper can be properly initialized", func() {
			databaseName := "suite-1-case-1"
			databaseCopy := database.DeepCopy()
			databaseCopy.SetName(databaseName)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, databaseCopy)
			err := kh.Cli.Create(ctx, databaseCopy)
			Expect(err).NotTo(HaveOccurred())
		})
	})
	Context("Test object creation", func() {
		It("Test that a new object cat be created", func() {
			secretName := "suite-2-test-1"
			secretCopy := secret.DeepCopy()
			secretCopy.SetName(secretName)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)
			res, err := kh.Create(ctx, secretCopy)
			Expect(err).To(BeNil())
			Expect(err).NotTo(HaveOccurred())

			actual := &corev1.Secret{}
			err = kh.Cli.Get(ctx, types.NamespacedName{Namespace: res.GetNamespace(), Name: res.GetName()}, actual)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual.GetName()).To(Equal(secretName))
		})
		It("Test an object that already exists", func() {
			secretName := "suite-2-test-2"
			secretCopy := secret.DeepCopy()
			secretCopy.SetName(secretName)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)
			err := kh.Cli.Create(ctx, secretCopy.DeepCopy())
			Expect(err).NotTo(HaveOccurred())

			actual, err := kh.Create(ctx, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual.GetName()).To(Equal(secretName))
		})
		It("Test an actual error while creating", func() {
			secretName := "suite-2-test-3"
			secretCopy := secret.DeepCopy()
			secretCopy.SetName(secretName)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)

			// Create an object, so it has resourceVersion set.
			// Then when creating the second time, there will be an error
			// "resourceVersion should not be set on objects to be created"
			err := kh.Cli.Create(ctx, secretCopy)
			Expect(err).NotTo(HaveOccurred())

			_, err = kh.Create(ctx, secretCopy)
			Expect(err).To(HaveOccurred())
		})
	})
	Context("Test object update", func() {
		It("Not used by any, no cleanup", func() {
			secretName := "suite-3-test-1"
			secretCopy := secret.DeepCopy()
			secretCopy.SetName(secretName)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)
			err := kh.Cli.Create(ctx, secretCopy.DeepCopy())
			Expect(err).NotTo(HaveOccurred())
			err = kh.Cli.Get(ctx, types.NamespacedName{Namespace: secretCopy.GetNamespace(), Name: secretName}, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			err = kh.Update(ctx, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			expectedLabels := map[string]string{
				consts.USED_BY_KIND_LABEL_KEY: database.GetObjectKind().GroupVersionKind().Kind,
				consts.USED_BY_NAME_LABEL_KEY: database.GetName(),
			}
			Expect(secretCopy.GetLabels()).To(Equal(expectedLabels))
			Expect(secretCopy.GetOwnerReferences()).To(BeEmpty())
		})

		It("Used by the caller, no cleanup", func() {
			secretName := "suite-3-test-2"
			usedByLabels := map[string]string{
				consts.USED_BY_KIND_LABEL_KEY: database.GetObjectKind().GroupVersionKind().Kind,
				consts.USED_BY_NAME_LABEL_KEY: database.GetName(),
			}
			secretCopy := secret.DeepCopy()
			secretCopy.SetLabels(usedByLabels)
			secretCopy.SetName(secretName)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)
			err := kh.Cli.Create(ctx, secretCopy.DeepCopy())
			Expect(err).NotTo(HaveOccurred())
			err = kh.Cli.Get(ctx, types.NamespacedName{Namespace: secretCopy.GetNamespace(), Name: secretName}, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			err = kh.Update(ctx, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			Expect(secretCopy.GetLabels()).To(Equal(usedByLabels))
			Expect(secretCopy.GetOwnerReferences()).To(BeEmpty())
		})
		It("Not used by any, with cleanup", func() {
			secretName := "suite-3-test-3"
			secretCopy := secret.DeepCopy()
			secretCopy.SetName(secretName)
			databaseCopy := database.DeepCopy()
			databaseCopy.Spec.Cleanup = true
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, databaseCopy)
			err := kh.Cli.Create(ctx, secretCopy.DeepCopy())
			Expect(err).NotTo(HaveOccurred())
			err = kh.Cli.Get(ctx, types.NamespacedName{Namespace: secretCopy.GetNamespace(), Name: secretName}, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			err = kh.Update(ctx, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			expectedOwnerReference := []metav1.OwnerReference{
				{
					APIVersion: database.APIVersion,
					Kind:       database.Kind,
					Name:       database.Name,
					UID:        database.UID,
				},
			}
			Expect(secretCopy.GetOwnerReferences()).To(Equal(expectedOwnerReference))
		})
		It("Used by the caller, with cleanup", func() {
			secretName := "suite-3-test-4"
			secretCopy := secret.DeepCopy()
			secretCopy.SetName(secretName)
			databaseCopy := database.DeepCopy()
			databaseCopy.Spec.Cleanup = true
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, databaseCopy)
			err := kh.Cli.Create(ctx, secretCopy.DeepCopy())
			Expect(err).NotTo(HaveOccurred())
			err = kh.Cli.Get(ctx, types.NamespacedName{Namespace: secretCopy.GetNamespace(), Name: secretName}, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			err = kh.Update(ctx, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			expectedOwnerReference := []metav1.OwnerReference{
				{
					APIVersion: database.APIVersion,
					Kind:       database.Kind,
					Name:       database.Name,
					UID:        database.UID,
				},
			}
			Expect(secretCopy.GetOwnerReferences()).To(Equal(expectedOwnerReference))
		})

		It("Not used by the caller, no cleanup", func() {
			secretName := "suite-3-test-5"
			usedByLabels := map[string]string{
				consts.USED_BY_KIND_LABEL_KEY: "test",
				consts.USED_BY_NAME_LABEL_KEY: "test",
			}
			secretCopy := secret.DeepCopy()
			secretCopy.SetLabels(usedByLabels)
			secretCopy.SetName(secretName)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)
			err := kh.Cli.Create(ctx, secretCopy.DeepCopy())
			Expect(err).NotTo(HaveOccurred())
			err = kh.Cli.Get(ctx, types.NamespacedName{Namespace: secretCopy.GetNamespace(), Name: secretName}, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			err = kh.Update(ctx, secretCopy)
			Expect(err).To(HaveOccurred())
		})
	})
	Context("Test create or update handler", func() {
		It("Create a new object", func() {
			secretName := "suite-4-test-1"
			secretCopy := secret.DeepCopy()
			secretCopy.SetName(secretName)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)
			err := kh.HandleCreateOrUpdate(ctx, secretCopy.DeepCopy())
			Expect(err).NotTo(HaveOccurred())
			usedByLabels := map[string]string{
				consts.USED_BY_KIND_LABEL_KEY: database.GetObjectKind().GroupVersionKind().Kind,
				consts.USED_BY_NAME_LABEL_KEY: database.GetName(),
			}
			err = kh.Cli.Get(ctx, types.NamespacedName{Namespace: secretCopy.GetNamespace(), Name: secretName}, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			Expect(secretCopy.Labels).To(Equal(usedByLabels))
		})
		It("Update an existing object", func() {
			secretName := "suite-4-test-2"
			secretCopy := secret.DeepCopy()
			secretCopy.SetName(secretName)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)
			err := kh.Cli.Create(ctx, secretCopy.DeepCopy())
			Expect(err).NotTo(HaveOccurred())
			err = kh.HandleCreateOrUpdate(ctx, secretCopy.DeepCopy())
			Expect(err).NotTo(HaveOccurred())
			usedByLabels := map[string]string{
				consts.USED_BY_KIND_LABEL_KEY: database.GetObjectKind().GroupVersionKind().Kind,
				consts.USED_BY_NAME_LABEL_KEY: database.GetName(),
			}
			err = kh.Cli.Get(ctx, types.NamespacedName{Namespace: secretCopy.GetNamespace(), Name: secretName}, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			Expect(secretCopy.Labels).To(Equal(usedByLabels))
		})
		It("Fail creation", func() {
			secretCopy := secret.DeepCopy()
			secretCopy.SetName("INVALID_NAME_!!@@")
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)
			err := kh.HandleCreateOrUpdate(ctx, secretCopy.DeepCopy())
			Expect(err).To(HaveOccurred())
		})
		It("Fail update", func() {
			secretName := "suite-4-test-4"
			secretCopy := secret.DeepCopy()
			secretCopy.SetName(secretName)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)
			err := kh.Cli.Create(ctx, secretCopy.DeepCopy())
			Expect(err).NotTo(HaveOccurred())
			invalidLabels := map[string]string{
				"___AAA": "!@#$%",
			}
			secretCopy.SetLabels(invalidLabels)
			err = kh.HandleCreateOrUpdate(ctx, secretCopy)
			Expect(err).To(HaveOccurred())
		})
	})
	Context("Test object removing", func() {
		It("Object with usedBy labels of the caller", func() {
			secretName := "suite-5-test-1"
			secretCopy := secret.DeepCopy()
			usedByLabels := map[string]string{
				consts.USED_BY_KIND_LABEL_KEY: database.GetObjectKind().GroupVersionKind().Kind,
				consts.USED_BY_NAME_LABEL_KEY: database.GetName(),
			}
			secretCopy.SetName(secretName)
			secretCopy.SetLabels(usedByLabels)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)
			err := kh.Cli.Create(ctx, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			err = kh.HandleDelete(ctx, secretCopy)
			Expect(err).NotTo(HaveOccurred())
		})
		It("UsedBy labels don't belong to the caller", func() {
			secretName := "suite-5-test-2"
			secretCopy := secret.DeepCopy()
			usedByLabels := map[string]string{
				consts.USED_BY_KIND_LABEL_KEY: "test",
				consts.USED_BY_NAME_LABEL_KEY: "test",
			}
			secretCopy.SetName(secretName)
			secretCopy.SetLabels(usedByLabels)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)
			err := kh.Cli.Create(ctx, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			err = kh.HandleDelete(ctx, secretCopy)
			Expect(err).To(HaveOccurred())
		})
		It("Fail update", func() {
			secretName := "suite-5-test-3"
			secretCopy := secret.DeepCopy()
			secretCopy.SetName(secretName)
			usedByLabels := map[string]string{
				consts.USED_BY_KIND_LABEL_KEY: "test",
				consts.USED_BY_NAME_LABEL_KEY: "test",
			}
			secretCopy.SetLabels(usedByLabels)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)
			err := kh.Cli.Create(ctx, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			invalidLabels := map[string]string{
				"INVALULID_LABELS": "!@#$!@#",
			}
			maps.Copy(usedByLabels, invalidLabels)
			secretCopy.SetLabels(usedByLabels)
			err = kh.HandleDelete(ctx, secretCopy)
			Expect(err).To(HaveOccurred())
		})
	})
	Context("Test the main handler", func() {
		It("Caller is removed", func() {
			secretName := "suite-6-test-1"
			secretCopy := secret.DeepCopy()
			secretCopy.SetName(secretName)
			databaseCopy := database.DeepCopy()
			databaseCopy.SetDeletionTimestamp(&metav1.Time{Time: time.Now()})

			usedByLabels := map[string]string{
				consts.USED_BY_KIND_LABEL_KEY: database.GetObjectKind().GroupVersionKind().Kind,
				consts.USED_BY_NAME_LABEL_KEY: database.GetName(),
			}

			secretCopy.SetLabels(usedByLabels)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, databaseCopy)
			err := kh.Cli.Create(ctx, secretCopy.DeepCopy())
			Expect(err).NotTo(HaveOccurred())
			err = kh.ModifyObject(ctx, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			Expect(secretCopy.GetLabels()).To(BeEmpty())
		})
		It("Caller is updated", func() {
			secretName := "suite-6-test-2"
			secretCopy := secret.DeepCopy()
			secretCopy.SetName(secretName)
			expectedUsedByLabels := map[string]string{
				consts.USED_BY_KIND_LABEL_KEY: database.GetObjectKind().GroupVersionKind().Kind,
				consts.USED_BY_NAME_LABEL_KEY: database.GetName(),
			}

			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)
			err := kh.ModifyObject(ctx, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			err = kh.Cli.Get(ctx, types.NamespacedName{Namespace: secretCopy.GetNamespace(), Name: secretCopy.GetName()}, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			Expect(secretCopy.GetLabels()).To(Equal(expectedUsedByLabels))
		})
	})
	Context("Test the value getter", func() {
		It("Get from Secret", func() {
			secretName := "suite-7-test-1"
			secretCopy := secret.DeepCopy()
			secretCopy.SetName(secretName)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)
			err := kh.Cli.Create(ctx, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			secretCopy.Data = map[string][]byte{
				"test": []byte("test"),
			}
			err = kh.Cli.Update(ctx, secretCopy)
			Expect(err).NotTo(HaveOccurred())

			val, err := kh.GetValueFrom(ctx, kube.SECRET, "default", secretName, "test")
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal("test"))
		})
		It("Get from ConfigMap", func() {
			cmName := "suite-7-test-2"
			cmCopy := configmap.DeepCopy()
			cmCopy.SetName(cmName)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)
			err := kh.Cli.Create(ctx, cmCopy)
			Expect(err).NotTo(HaveOccurred())
			cmCopy.Data = map[string]string{
				"test": "test",
			}
			err = kh.Cli.Update(ctx, cmCopy)
			Expect(err).NotTo(HaveOccurred())

			val, err := kh.GetValueFrom(ctx, kube.CONFIGMAP, "default", cmName, "test")
			Expect(err).NotTo(HaveOccurred())
			Expect(val).To(Equal("test"))
		})
		It("Get from Secret with error", func() {
			secretName := "suite-7-test-3"
			secretCopy := secret.DeepCopy()
			secretCopy.SetName(secretName)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)
			err := kh.Cli.Create(ctx, secretCopy)
			Expect(err).NotTo(HaveOccurred())
			secretCopy.Data = map[string][]byte{
				"test": []byte("test"),
			}
			err = kh.Cli.Update(ctx, secretCopy)
			Expect(err).NotTo(HaveOccurred())

			_, err = kh.GetValueFrom(ctx, kube.SECRET, "default", secretName, "dummy")
			Expect(err).To(HaveOccurred())
		})
		It("Get from ConfigMap with error", func() {
			cmName := "suite-7-test-4"
			cmCopy := configmap.DeepCopy()
			cmCopy.SetName(cmName)
			rec := events.NewFakeRecorder(1)
			kh := kube.NewKubeHelper(k8sClient, rec, database)
			err := kh.Cli.Create(ctx, cmCopy)
			Expect(err).NotTo(HaveOccurred())
			cmCopy.Data = map[string]string{
				"test": "test",
			}
			err = kh.Cli.Update(ctx, cmCopy)
			Expect(err).NotTo(HaveOccurred())

			_, err = kh.GetValueFrom(ctx, kube.CONFIGMAP, "default", cmName, "dummy")
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

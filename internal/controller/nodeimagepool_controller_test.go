/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sv1alpha1 "github.com/kenmoini/k8s-node-image-pool-operator/api/v1alpha1"
)

var _ = Describe("NodeImagePool Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		nodeimagepool := &k8sv1alpha1.NodeImagePool{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind NodeImagePool")
			err := k8sClient.Get(ctx, typeNamespacedName, nodeimagepool)
			if err != nil && errors.IsNotFound(err) {
				resource := &k8sv1alpha1.NodeImagePool{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &k8sv1alpha1.NodeImagePool{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance NodeImagePool")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &NodeImagePoolReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})

var _ = Describe("NodeMatches", func() {
	var testNode *corev1.Node

	BeforeEach(func() {
		testNode = &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-node",
				Labels: map[string]string{
					"node-role.kubernetes.io/worker": "",
					"kubernetes.io/arch":             "amd64",
					"kubernetes.io/os":               "linux",
					"region":                         "us-east-1",
				},
			},
		}
	})

	Context("with MatchLabels only", func() {
		It("should match when all labels match exactly", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchLabels: map[string]string{
					"node-role.kubernetes.io/worker": "",
				},
			}
			matches, err := NodeMatches(testNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeTrue())
		})

		It("should match with multiple labels", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchLabels: map[string]string{
					"node-role.kubernetes.io/worker": "",
					"kubernetes.io/arch":             "amd64",
				},
			}
			matches, err := NodeMatches(testNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeTrue())
		})

		It("should not match when label value differs", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchLabels: map[string]string{
					"node-role.kubernetes.io/worker": "true",
				},
			}
			matches, err := NodeMatches(testNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeFalse())
		})

		It("should not match when label key does not exist", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchLabels: map[string]string{
					"non-existent-label": "value",
				},
			}
			matches, err := NodeMatches(testNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeFalse())
		})

		It("should handle empty string label values", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchLabels: map[string]string{
					"node-role.kubernetes.io/worker": "",
				},
			}
			matches, err := NodeMatches(testNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeTrue())
		})
	})

	Context("with MatchExpressions only", func() {
		It("should match with In operator", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/arch",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"amd64", "arm64"},
					},
				},
			}
			matches, err := NodeMatches(testNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeTrue())
		})

		It("should not match with In operator when value not in list", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/arch",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"arm64", "s390x"},
					},
				},
			}
			matches, err := NodeMatches(testNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeFalse())
		})

		It("should match with NotIn operator", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/arch",
						Operator: corev1.NodeSelectorOpNotIn,
						Values:   []string{"arm64", "s390x"},
					},
				},
			}
			matches, err := NodeMatches(testNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeTrue())
		})

		It("should match with Exists operator", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/arch",
						Operator: corev1.NodeSelectorOpExists,
					},
				},
			}
			matches, err := NodeMatches(testNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeTrue())
		})

		It("should not match with DoesNotExist operator when label exists", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/arch",
						Operator: corev1.NodeSelectorOpDoesNotExist,
					},
				},
			}
			matches, err := NodeMatches(testNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeFalse())
		})
	})

	Context("with both MatchLabels and MatchExpressions", func() {
		It("should require both to match (AND logic)", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchLabels: map[string]string{
					"node-role.kubernetes.io/worker": "",
				},
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/arch",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"amd64", "arm64"},
					},
				},
			}
			matches, err := NodeMatches(testNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeTrue())
		})

		It("should not match when MatchLabels fails but MatchExpressions passes", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchLabels: map[string]string{
					"node-role.kubernetes.io/master": "",
				},
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/arch",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"amd64", "arm64"},
					},
				},
			}
			matches, err := NodeMatches(testNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeFalse())
		})

		It("should not match when MatchExpressions fails but MatchLabels passes", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchLabels: map[string]string{
					"node-role.kubernetes.io/worker": "",
				},
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/arch",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"arm64", "s390x"},
					},
				},
			}
			matches, err := NodeMatches(testNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeFalse())
		})

		It("should match with multiple labels and expressions", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchLabels: map[string]string{
					"node-role.kubernetes.io/worker": "",
					"kubernetes.io/os":               "linux",
				},
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/arch",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"amd64", "arm64"},
					},
					{
						Key:      "region",
						Operator: corev1.NodeSelectorOpExists,
					},
				},
			}
			matches, err := NodeMatches(testNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeTrue())
		})
	})

	Context("with empty selectors", func() {
		It("should match all nodes when both selectors are empty", func() {
			pool := k8sv1alpha1.CachePools{
				Name:             "test-pool",
				MatchLabels:      map[string]string{},
				MatchExpressions: []corev1.NodeSelectorRequirement{},
			}
			matches, err := NodeMatches(testNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeTrue())
		})

		It("should match all nodes when both selectors are nil", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
			}
			matches, err := NodeMatches(testNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeTrue())
		})
	})

	Context("with edge cases", func() {
		It("should handle node with no labels", func() {
			emptyNode := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "empty-node",
					Labels: map[string]string{},
				},
			}
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchLabels: map[string]string{
					"some-label": "value",
				},
			}
			matches, err := NodeMatches(emptyNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeFalse())
		})

		It("should match when expression checks for non-existence", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "non-existent-label",
						Operator: corev1.NodeSelectorOpDoesNotExist,
					},
				},
			}
			matches, err := NodeMatches(testNode, pool)
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeTrue())
		})
	})
})

var _ = Describe("Helper Functions", func() {
	Describe("hasAnySelector", func() {
		It("should return true when MatchLabels is defined", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchLabels: map[string]string{
					"key": "value",
				},
			}
			Expect(hasAnySelector(pool)).To(BeTrue())
		})

		It("should return true when MatchExpressions is defined", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "key",
						Operator: corev1.NodeSelectorOpExists,
					},
				},
			}
			Expect(hasAnySelector(pool)).To(BeTrue())
		})

		It("should return true when both are defined", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchLabels: map[string]string{
					"key": "value",
				},
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "key",
						Operator: corev1.NodeSelectorOpExists,
					},
				},
			}
			Expect(hasAnySelector(pool)).To(BeTrue())
		})

		It("should return false when both are empty", func() {
			pool := k8sv1alpha1.CachePools{
				Name:             "test-pool",
				MatchLabels:      map[string]string{},
				MatchExpressions: []corev1.NodeSelectorRequirement{},
			}
			Expect(hasAnySelector(pool)).To(BeFalse())
		})

		It("should return false when both are nil", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
			}
			Expect(hasAnySelector(pool)).To(BeFalse())
		})
	})

	Describe("validateCachePoolSelector", func() {
		It("should return nil when selectors are defined", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
				MatchLabels: map[string]string{
					"key": "value",
				},
			}
			err := validateCachePoolSelector(pool)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error when no selectors are defined", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "test-pool",
			}
			err := validateCachePoolSelector(pool)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no selectors defined"))
		})

		It("should include pool name in error message", func() {
			pool := k8sv1alpha1.CachePools{
				Name: "my-specific-pool",
			}
			err := validateCachePoolSelector(pool)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("my-specific-pool"))
		})
	})
})

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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1alpha1 "github.com/kenmoini/k8s-node-image-pool-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

// NodeImagePoolReconciler reconciles a NodeImagePool object
type NodeImagePoolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NodeMatches(node *corev1.Node, pool k8sv1alpha1.CachePools) (bool, error) {
	// 1. Create a selector
	selector := labels.NewSelector()

	// 2. Add MatchLabels as requirements (using Equals operator)
	for key, value := range pool.MatchLabels {
		req, err := labels.NewRequirement(key, selection.Equals, []string{value})
		if err != nil {
			return false, fmt.Errorf("invalid matchLabel %s=%s: %w", key, value, err)
		}
		selector = selector.Add(*req)
	}

	// 3. Add MatchExpressions as requirements
	for _, expr := range pool.MatchExpressions {
		// Map NodeSelectorOperator to selection.Operator
		var op selection.Operator
		switch expr.Operator {
		case corev1.NodeSelectorOpIn:
			op = selection.In
		case corev1.NodeSelectorOpNotIn:
			op = selection.NotIn
		case corev1.NodeSelectorOpExists:
			op = selection.Exists
		case corev1.NodeSelectorOpDoesNotExist:
			op = selection.DoesNotExist
		case corev1.NodeSelectorOpGt:
			op = selection.GreaterThan
		case corev1.NodeSelectorOpLt:
			op = selection.LessThan
		default:
			return false, fmt.Errorf("unsupported operator %s for key %s", expr.Operator, expr.Key)
		}

		req, err := labels.NewRequirement(expr.Key, op, expr.Values)
		if err != nil {
			return false, fmt.Errorf("invalid matchExpression for key %s: %w", expr.Key, err)
		}
		selector = selector.Add(*req)
	}

	// 4. Evaluate against node labels (both MatchLabels and MatchExpressions must match - AND logic)
	return selector.Matches(labels.Set(node.Labels)), nil
}

// hasAnySelector checks if a CachePool has any selectors defined
func hasAnySelector(pool k8sv1alpha1.CachePools) bool {
	return len(pool.MatchLabels) > 0 || len(pool.MatchExpressions) > 0
}

// validateCachePoolSelector validates that a CachePool has valid selectors
func validateCachePoolSelector(pool k8sv1alpha1.CachePools) error {
	if !hasAnySelector(pool) {
		return fmt.Errorf("cachePool '%s' has no selectors defined", pool.Name)
	}
	return nil
}

// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=k8s.armillary.io,resources=nodeimagepools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.armillary.io,resources=nodeimagepools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.armillary.io,resources=nodeimagepools/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NodeImagePool object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *NodeImagePoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	globalLog = ctrl.Log.WithName("k8s-node-image-pool-controller")

	nodeImagePool := &k8sv1alpha1.NodeImagePool{}
	globalLog.Info("Reconciling NodeImagePool", "name", req.NamespacedName)

	if err := r.Get(ctx, req.NamespacedName, nodeImagePool); err != nil {
		globalLog.Error(err, "unable to fetch NodeImagePool")
		// we'll ignore not-found errors, since they can't be fixed by an immediate requeue
		return ctrl.Result{}, client.IgnoreNotFound(err)
	} else {
		globalLog.Info("Fetched NodeImagePool", "spec", nodeImagePool.Spec)
		logger := ctrl.Log.WithName(nodeImagePool.Name)

		// ===========================================================================
		// Basic validation checks
		// ===========================================================================

		validCachePools := []k8sv1alpha1.CachePools{}
		validConsumerPools := []k8sv1alpha1.CachePools{}

		// Get the list of nodes in the cluster
		nodes := &corev1.NodeList{}
		listOpts := []client.ListOption{}

		// Get all the Nodes
		if err := r.List(ctx, nodes, listOpts...); err != nil {
			logger.Error(err, "Failed to list nodes")
			return ctrl.Result{}, err
		}
		logger.Info("Total nodes in cluster", "count", len(nodes.Items))

		// Check to see if CachePools are defined and validate nodes against them
		if len(nodeImagePool.Spec.CachePools) == 0 {
			logger.Info("No CachePools defined, skipping reconciliation")
			return ctrl.Result{}, nil
		} else {
			logger.Info("CachePools defined", "count", len(nodeImagePool.Spec.CachePools))
			for _, node := range nodes.Items {
				// Check to see if any nodes match the CachePools selectors
				// Loop through each CachePool and match against node labels
				for _, cachePool := range nodeImagePool.Spec.CachePools {
					// Validate that the CachePool has selectors defined
					if err := validateCachePoolSelector(cachePool); err != nil {
						logger.Info("Skipping CachePool with no selectors", "cachePool", cachePool.Name)
						continue
					}

					matches := false
					var err error
					matches, err = NodeMatches(&node, cachePool)
					if err != nil {
						logger.Error(err, "Error matching node to CachePool", "node", node.Name, "cachePool", cachePool.Name)
						continue
					}

					if matches {
						logger.Info("Node matches CachePool", "node", node.Name, "cachePool", cachePool.Name)
						// Add to validCachePools if not already present
						found := false
						for _, vcp := range validCachePools {
							if vcp.Name == cachePool.Name {
								found = true
								break
							}
						}
						if !found {
							validCachePools = append(validCachePools, cachePool)
						}
					} else {
						logger.Info("Node does not match CachePool", "node", node.Name, "cachePool", cachePool.Name)
					}
				}
			}
		}

		// Check to see if CacheConsumers are defined and validate nodes against them
		if len(nodeImagePool.Spec.CacheConsumers) == 0 {
			logger.Info("No CacheConsumers defined, skipping reconciliation")
			return ctrl.Result{}, nil
		} else {
			logger.Info("CacheConsumers defined", "count", len(nodeImagePool.Spec.CacheConsumers))
			// Check to see if any nodes match the CacheConsumers selectors
			// Loop through each CacheConsumer and match against node labels
			for _, node := range nodes.Items {
				for _, cacheConsumer := range nodeImagePool.Spec.CacheConsumers {
					// Validate that the CacheConsumer has selectors defined
					if err := validateCachePoolSelector(cacheConsumer); err != nil {
						logger.Info("Skipping CacheConsumer with no selectors", "cacheConsumer", cacheConsumer.Name)
						continue
					}

					matches := false
					var err error
					matches, err = NodeMatches(&node, cacheConsumer)
					if err != nil {
						logger.Error(err, "Error matching node to CacheConsumer", "node", node.Name, "cacheConsumer", cacheConsumer.Name)
						continue
					}

					if matches {
						logger.Info("Node matches CacheConsumer", "node", node.Name, "cacheConsumer", cacheConsumer.Name)
						// Add to validConsumerPools if not already present
						found := false
						for _, vcp := range validConsumerPools {
							if vcp.Name == cacheConsumer.Name {
								found = true
								break
							}
						}
						if !found {
							validConsumerPools = append(validConsumerPools, cacheConsumer)
						}
					} else {
						logger.Info("Node does not match CacheConsumer", "node", node.Name, "cacheConsumer", cacheConsumer.Name)
					}
				}
			}
		}

		// Discount double-checking - ensure that we have at least one valid CachePool and CacheConsumer
		if len(validCachePools) == 0 {
			logger.Info("No valid CachePools found after evaluation, skipping reconciliation")
			return ctrl.Result{}, nil
		}
		if len(validConsumerPools) == 0 {
			logger.Info("No valid CacheConsumers found after evaluation, skipping reconciliation")
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeImagePoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.NodeImagePool{}).
		Named("nodeimagepool").
		Complete(r)
}

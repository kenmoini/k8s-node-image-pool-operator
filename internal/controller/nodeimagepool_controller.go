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
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1alpha1 "github.com/kenmoini/k8s-node-image-pool-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeImagePoolReconciler reconciles a NodeImagePool object
type NodeImagePoolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=namespaces;configmaps,verbs=get;create;update;list;watch
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
// nolint: gocyclo
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

		// Create mappings of CachePool and CacheConsumer names to their hosts for easy lookup later
		cachePoolMapping := make(map[string][]string)
		cachePoolHostIPMapping := make(map[string]string)
		consumerPoolMapping := make(map[string][]string)

		// Get the list of nodes in the cluster
		nodes := &corev1.NodeList{}
		listOpts := []client.ListOption{}

		// Get all the Nodes
		if err := r.List(ctx, nodes, listOpts...); err != nil {
			logger.Error(err, "Failed to list nodes")
			return ctrl.Result{}, err
		}
		logger.Info("Total nodes in cluster", "count", len(nodes.Items))

		// Make sure mirrorFiltering is defined
		if len(nodeImagePool.Spec.MirrorFiltering) == 0 {
			logger.Info("No MirrorFiltering defined, skipping reconciliation")
			return ctrl.Result{}, nil
		} else {
			logger.Info("MirrorFiltering defined", "mirrors", nodeImagePool.Spec.MirrorFiltering)
		}

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
						// Add to cachePoolMapping for easy lookup later if it doesn't already exist
						if _, exists := cachePoolMapping[cachePool.Name]; !exists {
							cachePoolMapping[cachePool.Name] = []string{}
						}
						// Append the node name to the list for this CachePool if not already present
						if !containsString(cachePoolMapping[cachePool.Name], node.Name) {
							cachePoolMapping[cachePool.Name] = append(cachePoolMapping[cachePool.Name], node.Name)
						}
						// Store the node's InternalIP for use in the ConfigMap
						for _, addr := range node.Status.Addresses {
							if addr.Type == corev1.NodeInternalIP {
								cachePoolHostIPMapping[node.Name] = addr.Address
								break
							}
						}
						logger.Info("Node IP for CachePool", "node", node.Name, "ip", cachePoolHostIPMapping[node.Name])
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

					// Add to consumerPoolMapping for easy lookup later if it doesn't already exist
					if _, exists := consumerPoolMapping[cacheConsumer.Name]; !exists {
						consumerPoolMapping[cacheConsumer.Name] = []string{}
					}
					// Append the node name to the list for this CacheConsumer if not already present
					if !containsString(consumerPoolMapping[cacheConsumer.Name], node.Name) {
						consumerPoolMapping[cacheConsumer.Name] = append(consumerPoolMapping[cacheConsumer.Name], node.Name)
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

		// Print out the cachePoolMapping and consumerPoolMapping for debugging
		logger.Info("Valid CachePools and their nodes", "mapping", cachePoolMapping)
		logger.Info("Valid CacheConsumers and their nodes", "mapping", consumerPoolMapping)

		// ===========================================================================
		// Namespace creation
		// ===========================================================================
		// Create the namespace if it doesn't exist

		if err := setupNamespace(ctx, r.Client, logger); err != nil {
			logger.Error(err, "Failed to setup namespace", "namespace", DefaultNamespace)
			return ctrl.Result{}, err
		}

		// ==========================================================================
		// ConfigMap creation for registry configuration
		// ==========================================================================

		// The ConfigMap has a key per host in the cachePoolMapping, with the value being
		// the registry configuration for that host.
		// The configuration has the mirror specification for each cachePool host, unless
		// the cachePool host is the same as the current cacheConsumer host, in which case
		// it is skipped to avoid circular references.

		// Create the ConfigMap data structure
		// Loop through the consumerPoolMapping then loop through each Node in that pool
		configMapData := map[string]string{}
		for consumerPoolName, consumerNodes := range consumerPoolMapping {
			logger.Info("Processing consumerPool", "consumerPool", consumerPoolName, "nodes", consumerNodes)
			for _, consumerNodeName := range consumerNodes {
				configMapData[consumerNodeName] = ""
				// Loop through the cachePoolMapping to add mirrors, skipping if the cachePool is the same as the consumerPool
				configData := ""
				configMirrorData := ""
				for cachePoolName, cachePoolNodes := range cachePoolMapping {
					for _, cachePoolNodeName := range cachePoolNodes {
						if cachePoolNodeName == consumerNodeName {
							logger.Info("Skipping mirror for cachePool as it matches consumer node", "consumerNode", consumerNodeName, "cachePool", cachePoolName)
							continue
						} else {
							configMirrorData += "  [[registry.mirror]]\n"
							configMirrorData += "    # Mirror for cachePool " + cachePoolName + " on node " + cachePoolNodeName + "\n"
							configMirrorData += "    location = \"" + cachePoolHostIPMapping[cachePoolNodeName] + ":" + strconv.Itoa(DefaultHostPort) + "\"\n"
						}

					}
				}
				// Loop through the mirrorFiltering to add the filtered mirrors
				for _, mirrorHost := range nodeImagePool.Spec.MirrorFiltering {
					configData += "[[registry]]\n"
					configData += "  prefix = \"\"\n"
					configData += "  location = \"" + mirrorHost + "\"\n"
					configData += configMirrorData
				}
				configMapData[consumerNodeName] = configData
			}
		}

		configMap := &corev1.ConfigMap{}
		configMapName := DefaultMirrorConfigMapName
		err = r.Get(ctx, client.ObjectKey{Name: configMapName, Namespace: DefaultNamespace}, configMap)
		if err != nil {
			logger.Info("ConfigMap does not exist, creating", "configMap", configMapName)
			configMap = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMapName,
					Namespace: DefaultNamespace,
				},
				Data: configMapData,
			}

			if err := r.Create(ctx, configMap); err != nil {
				logger.Error(err, "Failed to create ConfigMap", "configMap", configMapName)
				return ctrl.Result{}, err
			}
			logger.Info("ConfigMap created successfully", "configMap", configMapName)
		} else {
			// Check to see if the data needs to be updated
			needsUpdate := false
			for key, value := range configMapData {
				if existingValue, exists := configMap.Data[key]; !exists || existingValue != value {
					needsUpdate = true
					break
				}
			}
			if needsUpdate {
				logger.Info("ConfigMap data needs update, updating", "configMap", configMapName)
				configMap.Data = configMapData
				if err := r.Update(ctx, configMap); err != nil {
					logger.Error(err, "Failed to update ConfigMap", "configMap", configMapName)
					return ctrl.Result{}, err
				}
				logger.Info("ConfigMap updated successfully", "configMap", configMapName)
			} else {
				logger.Info("ConfigMap data is up-to-date", "configMap", configMapName)
			}
		}

		// ==========================================================================
		// DaemonSet creation for registry on each node
		// ==========================================================================

		// Loop through the validCachePools and create a DaemonSet for each one if it doesn't already exist
		for _, cachePool := range validCachePools {
			logger.Info("Processing DaemonSet for CachePool", "cachePool", cachePool.Name)
			daemonSet := createDaemonSet(cachePool)

			// Check if the DaemonSet already exists
			existingDaemonSet := &appsv1.DaemonSet{}
			err := r.Get(ctx, client.ObjectKey{Name: daemonSet.Name, Namespace: DefaultNamespace}, existingDaemonSet)
			if err != nil {
				logger.Info("DaemonSet does not exist, creating", "daemonSet", daemonSet.Name)
				if err := r.Create(ctx, daemonSet); err != nil {
					logger.Error(err, "Failed to create DaemonSet", "daemonSet", daemonSet.Name)
					return ctrl.Result{}, err
				}
				logger.Info("DaemonSet created successfully", "daemonSet", daemonSet.Name)
			} else {
				// Update the DaemonSet anyways
				existingDaemonSet.Spec = daemonSet.Spec
				if err := r.Update(ctx, existingDaemonSet); err != nil {
					logger.Error(err, "Failed to update DaemonSet", "daemonSet", daemonSet.Name)
					return ctrl.Result{}, err
				}
				logger.Info("DaemonSet updated successfully", "daemonSet", daemonSet.Name)
			}

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

// containsString checks if a string slice contains a specific string
func containsString(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

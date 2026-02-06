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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodeImagePoolSpec defines the desired state of NodeImagePool.
type NodeImagePoolSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// MirrorFiltering provides a way to only cache/mirror certain endpoints
	MirrorFiltering []string `json:"mirrorFiltering,omitempty"`

	// CachePools allows for selecting nodes that the local cache/mirror will run on
	CachePools []CachePools `json:"cachePools,omitempty"`

	// CacheConsumers allows for selecting nodes that will use the local cache/mirror
	CacheConsumers []CachePools `json:"cacheConsumers,omitempty"`

	// NodeAuthentication provides credentials for nodes to access the cache/mirror
	NodeAuthentication NodeAuthentication `json:"nodeAuthentication,omitempty"`

	// ExternalAccess enables access to the cache/mirror from outside the cluster
	ExternalAccess ExternalAccess `json:"externalAccess,omitempty"`
}

// CachePools defines the node selection for caching/mirroring
type CachePools struct {
	// Name is a human readable name for the pool
	Name string `json:"name,omitempty"`

	// MatchLabels selects nodes based on labels
	MatchLabels map[string]string `json:"matchLabels,omitempty"`

	// MatchExpressions selects nodes based on label expressions
	MatchExpressions []metav1.LabelSelectorRequirement `json:"matchExpressions,omitempty"`

	// MatchExpressions []corev1.NodeSelectorRequirement `json:"matchExpressions,omitempty"`

	// Tolerations allows scheduling on tainted nodes
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// NodeAuthentication provides credentials for nodes to access the cache/mirror
type NodeAuthentication struct {
	// Provider indicates the authentication provider type.  Valid options are "None", "ServiceAccount", and "OIDC".	Default is "None".
	//
	// +kubebuilder:validation:Enum=None;ServiceAccount;OIDC
	// +kubebuilder:default=None
	Provider string `json:"provider,omitempty"`

	// ServiceAccount and OIDC settings would go here - to be implemented in future versions
}

// ExternalAccess enables access to the cache/mirror from outside the cluster
type ExternalAccess struct {
	// Enabled indicates whether external access is enabled
	Enabled bool `json:"enabled,omitempty"`

	// Ingress settings for management of entrypoints
	Ingress ExternalAccessIngress `json:"ingress,omitempty"`

	// AuthProvider indicates the external access authentication provider type.  Valid options are "None", "Basic", and "OIDC".	Default is "None".
	//
	// +kubebuilder:validation:Enum=None;Basic;OIDC
	// +kubebuilder:default=None
	AuthProvider string `json:"authProvider,omitempty"`

	// Basic and OIDC settings would go here - to be implemented in future versions
}

// ExternalAccessIngress defines ingress settings for external access
type ExternalAccessIngress struct {
	// Type indicates the Ingress type.  Valid options are "None", "LoadBalancer", "Ingress", and "OpenShiftRoute".	Default is "None".
	//
	// +kubebuilder:validation:Enum=None;LoadBalancer;Ingress;OpenShiftRoute
	// +kubebuilder:default=None
	Type string `json:"type,omitempty"`
}

// NodeImagePoolStatus defines the observed state of NodeImagePool.
type NodeImagePoolStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// NodeImagePool is the Schema for the nodeimagepools API.
type NodeImagePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeImagePoolSpec   `json:"spec,omitempty"`
	Status NodeImagePoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NodeImagePoolList contains a list of NodeImagePool.
type NodeImagePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeImagePool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeImagePool{}, &NodeImagePoolList{})
}

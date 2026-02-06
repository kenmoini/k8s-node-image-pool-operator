package controller

import (
	k8sv1alpha1 "github.com/kenmoini/k8s-node-image-pool-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createDaemonSet(cachePool k8sv1alpha1.CachePools) *appsv1.DaemonSet {
	// Implementation for creating a DaemonSet for the given CachePool
	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cachePool.Name + "-daemonset",
			Namespace: DefaultNamespace,
			Labels: map[string]string{
				"app": DefaultAppLabelValue,
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": DefaultAppLabelValue,
				},
				// MatchLabels:      cachePool.MatchLabels,
				// MatchExpressions: cachePool.MatchExpressions,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": DefaultAppLabelValue,
					},
				},
				Spec: v1.PodSpec{
					Tolerations: cachePool.Tolerations,
					Containers: []v1.Container{
						{
							Name:  "registry-container",
							Image: DefaultRegistryContainerImage,
							Ports: []v1.ContainerPort{
								{
									ContainerPort: DefaultHostPort,
								},
							},
						},
					},
				},
			},
			// DaemonSet spec details would go here
		},
	}
	return daemonSet
}

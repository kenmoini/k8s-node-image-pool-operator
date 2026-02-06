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
					HostNetwork: true,
					SecurityContext: &v1.PodSecurityContext{
						RunAsUser: func() *int64 {
							v := int64(0)
							return &v
						}(),
					},
					Volumes: []v1.Volume{
						{
							Name: "mirror-config-volume",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: DefaultMirrorConfigMapName,
									},
								},
							},
						},
						{
							Name: "mirror-config",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/etc/containers/registries.conf.d/99-node-image-pool.conf",
									Type: func() *v1.HostPathType {
										hostPathType := v1.HostPathFileOrCreate
										return &hostPathType
									}(),
								},
							},
						},
					},
					Containers: []v1.Container{
						{
							Name:  "registry-container",
							Image: DefaultRegistryContainerImage,
							Ports: []v1.ContainerPort{
								{
									ContainerPort: DefaultHostPort,
								},
							},
							ImagePullPolicy: v1.PullAlways,
							Env: []v1.EnvVar{
								{
									Name: "MY_NODE_NAME",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name: "MY_POD_NAME",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "MY_POD_NAMESPACE",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name: "MY_POD_IP",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name: "MY_POD_SERVICE_ACCOUNT",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "spec.serviceAccountName",
										},
									},
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "mirror-config-volume",
									MountPath: "/cacheConsumers/" + DefaultMirrorConfigMapName,
								},
								{
									Name:      "mirror-config",
									MountPath: "/etc/containers/registries.conf.d/99-node-image-pool.conf",
								},
							},
						},
					},
				},
			},
		},
	}
	return daemonSet
}

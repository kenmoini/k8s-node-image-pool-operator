package controller

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func setupNamespace(ctx context.Context, r client.Client, logger logr.Logger) error {
	namespace := &corev1.Namespace{}
	err := r.Get(ctx, client.ObjectKey{Name: DefaultNamespace}, namespace)
	if err != nil {
		logger.Info("Namespace does not exist, creating", "namespace", DefaultNamespace)
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: DefaultNamespace,
			},
		}
		if err := r.Create(ctx, namespace); err != nil {
			logger.Error(err, "Failed to create namespace", "namespace", DefaultNamespace)
			return err
		}
		logger.Info("Namespace created successfully", "namespace", DefaultNamespace)
	} else {
		logger.Info("Namespace already exists", "namespace", DefaultNamespace)
	}
	return nil
}

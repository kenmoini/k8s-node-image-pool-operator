package controller

import (
	"fmt"

	k8sv1alpha1 "github.com/kenmoini/k8s-node-image-pool-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

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
		case metav1.LabelSelectorOpIn:
			op = selection.In
		case metav1.LabelSelectorOpNotIn:
			op = selection.NotIn
		case metav1.LabelSelectorOpExists:
			op = selection.Exists
		case metav1.LabelSelectorOpDoesNotExist:
			op = selection.DoesNotExist
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

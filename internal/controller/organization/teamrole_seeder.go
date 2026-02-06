// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
)

var defaultTeamRoles = map[string]greenhousev1alpha1.TeamRoleSpec{
	"cluster-admin": {
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{"*"},
			Resources: []string{"*"},
			Verbs:     []string{"*"},
		}},
	},
	"cluster-viewer": {
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{"*"},
			Resources: []string{"*"},
			Verbs:     []string{"get", "list", "watch"},
		}},
		Labels: map[string]string{
			"greenhouse.sap/aggregate-to-developer": "true",
		},
	},
	"cluster-developer": {
		AggregationRule: &rbacv1.AggregationRule{
			ClusterRoleSelectors: []metav1.LabelSelector{{
				MatchLabels: map[string]string{
					"greenhouse.sap/aggregate-to-developer": "true",
				},
			}},
		},
	},
	"application-developer": {
		Labels: map[string]string{
			"greenhouse.sap/aggregate-to-developer": "true",
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			// no "pods/exec" to prevent privilege escalation through pod service accounts
			Resources: []string{"pods", "pods/portforward", "pods/eviction", "pods/proxy", "pods/log", "pods/status"},
			Verbs:     []string{"*"},
		}, {
			APIGroups: []string{"apps"},
			Resources: []string{"deployments/scale", "statefulsets/scale"},
			Verbs:     []string{"patch"},
		},
		},
	},
	"node-maintainer": {
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"nodes"},
			Verbs:     []string{"get", "patch"},
		}},
	},
	"namespace-creator": {
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"namespaces"},
			Verbs:     []string{"*"},
		}},
	},
}

func (r *OrganizationReconciler) reconcileDefaultTeamRoles(ctx context.Context, org *greenhousev1alpha1.Organization) error {
	for name, teamRoleSpec := range defaultTeamRoles {
		var tr = new(greenhousev1alpha1.TeamRole)
		tr.Name = name
		tr.Namespace = org.GetName()

		result, err := clientutil.CreateOrPatch(ctx, r.Client, tr, func() error {
			tr.Spec = teamRoleSpec
			return controllerutil.SetOwnerReference(org, tr, r.Scheme())
		})
		if err != nil {
			return err
		}
		switch result {
		case clientutil.OperationResultCreated:
			log.FromContext(ctx).Info("created team role", "namespace", tr.GetNamespace(), "name", tr.GetName())
			r.recorder.Eventf(org, tr, corev1.EventTypeNormal, "CreatedTeamRole", "reconciling Organization", "Created team role %s/%s", tr.GetNamespace(), tr.GetName())
		case clientutil.OperationResultUpdated:
			log.FromContext(ctx).Info("updated team role", "namespace", tr.GetNamespace(), "name", tr.GetName())
			r.recorder.Eventf(org, tr, corev1.EventTypeNormal, "UpdatedTeamRole", "reconciling Organization", "Updated team role %s/%s", tr.GetNamespace(), tr.GetName())
		}
	}
	return nil
}

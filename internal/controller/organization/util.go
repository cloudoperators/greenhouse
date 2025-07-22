// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

func listOrganizationsAsReconcileRequests(ctx context.Context, c client.Client, listOpts ...client.ListOption) []ctrl.Request {
	var organizationList = new(greenhousev1alpha1.OrganizationList)
	if err := c.List(ctx, organizationList, listOpts...); err != nil {
		return nil
	}
	res := make([]ctrl.Request, len(organizationList.Items))
	for idx, organization := range organizationList.Items {
		res[idx] = ctrl.Request{NamespacedName: types.NamespacedName{Name: organization.Name, Namespace: organization.Namespace}}
	}
	return res
}

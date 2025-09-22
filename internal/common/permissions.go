// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"

	authorizationv1 "k8s.io/api/authorization/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Permission defines a Kubernetes action on a resource.
type Permission struct {
	Name     string
	Verb     string
	APIGroup string
	Resource string
}

func (p Permission) String() string {
	return fmt.Sprintf("%s %s/%s", p.Verb, p.APIGroup, p.Resource)
}

var (
	// greenhousePermissions lists perms for the greenhouse cluster.
	greenhousePermissions = []Permission{
		{Name: "createCluster", Verb: "create", APIGroup: "greenhouse.sap", Resource: "clusters"},
		{Name: "deleteCluster", Verb: "delete", APIGroup: "greenhouse.sap", Resource: "clusters"},
		{Name: "updateCluster", Verb: "update", APIGroup: "greenhouse.sap", Resource: "clusters"},
		{Name: "patchCluster", Verb: "patch", APIGroup: "greenhouse.sap", Resource: "clusters"},
		{Name: "createSecret", Verb: "create", APIGroup: "", Resource: "secrets"},
		{Name: "updateSecret", Verb: "update", APIGroup: "", Resource: "secrets"},
		{Name: "patchSecret", Verb: "patch", APIGroup: "", Resource: "secrets"},
	}
	// clientClusterPermissions lists perms for the customer cluster.
	clientClusterPermissions = []Permission{
		{Name: "clusterAdmin", Verb: "*", APIGroup: "*", Resource: "*"},
	}
)

// CheckGreenhousePermission returns names of missing greenhouse permissions for the user.
func CheckGreenhousePermission(ctx context.Context, kubeClient client.Client, user, namespace string) (missingPermission []Permission) {
	return checkPermissionMap(ctx, kubeClient, greenhousePermissions, user, namespace)
}

// CheckClientClusterPermission returns names of missing client-cluster permissions.
func CheckClientClusterPermission(ctx context.Context, kubeClient client.Client, user, namespace string) (missingPermission []Permission) {
	return checkPermissionMap(ctx, kubeClient, clientClusterPermissions, user, namespace)
}

func checkPermissionMap(ctx context.Context, kubeClient client.Client, permissionMap []Permission, user, namespace string) (missingPermission []Permission) {
	for _, permission := range permissionMap {
		if !canI(ctx, kubeClient, namespace, user, permission) {
			missingPermission = append(missingPermission, permission)
		}
	}
	return missingPermission
}

func canI(ctx context.Context, kubeClient client.Client, user, namespace string, permission Permission) bool {
	if user == "" {
		accessReview := &authorizationv1.SelfSubjectAccessReview{
			Spec: authorizationv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Namespace: namespace,
					Verb:      permission.Verb,
					Group:     permission.APIGroup,
					Resource:  permission.Resource,
				},
			},
		}

		return kubeClient.Create(ctx, accessReview) == nil && accessReview.Status.Allowed
	}

	accessReview := &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      permission.Verb,
				Group:     permission.APIGroup,
				Resource:  permission.Resource,
			},
			User: user,
		},
	}

	return kubeClient.Create(ctx, accessReview) == nil && accessReview.Status.Allowed
}

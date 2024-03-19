// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package apis

import corev1 "k8s.io/api/core/v1"

const (
	// GroupName for greenhouse API resources.
	GroupName = "greenhouse.sap"

	// FinalizerCleanupHelmRelease is used to invoke the Helm release cleanup logic.
	FinalizerCleanupHelmRelease = "greenhouse.sap/helm"

	// FinalizerCleanupCluster is used to invoke the cleanup of a registered cluster.
	FinalizerCleanupCluster = "greenhouse.sap/cluster"

	// FinalizerCleanupPropagatedResource is used to invoke the cleanup of remote resources.
	FinalizerCleanupPropagatedResource = "greenhouse.sap/propagatedResource"

	// SecretTypeKubeConfig specifies a secret containing the kubeconfig for a cluster.
	SecretTypeKubeConfig corev1.SecretType = "greenhouse.sap/kubeconfig"

	// KubeConfigKey is the key for the user-provided kubeconfig in the secret of type greenhouse.sap/kubeconfig.
	KubeConfigKey = "kubeconfig"

	// GreenHouseKubeConfigKey is the key for the kubeconfig in the secret of type greenhouse.sap/kubeconfig used by Greenhouse.
	// This kubeconfig should be used by Greenhouse controllers and their kubernetes clients to access the remote cluster.
	GreenHouseKubeConfigKey = "greenhousekubeconfig"

	// HeadscalePreAuthKey is the key for the Headscale pre-authentication key in a secret of type greenhouse.sap/kubeconfig.
	HeadscalePreAuthKey = "headscalePreAuthKey"

	// LabelKeyPlugin is used to identify corresponding Plugin for the resource.
	LabelKeyPlugin = "greenhouse.sap/plugin"

	// LabelKeyCluster is used to identify corresponding Cluster for the resource.
	LabelKeyCluster = "greenhouse.sap/cluster"

	// HeadScaleKey is the key for the Headscale client deployment
	HeadScaleKey = "greenhouse.sap/headscale"

	// LabelAccessMode is used to force the access mode to headscale for a cluster.
	LabelAccessMode = "greenhouse.sap/access-mode"

	// LabelKeyExposeService is applied to services that are part of a plugins Helm chart to expose them via the central Greenhouse infrastructure.
	LabelKeyExposeService = "greenhouse.sap/expose"
)

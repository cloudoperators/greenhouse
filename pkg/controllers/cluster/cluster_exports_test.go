// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	headscalev1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ExportServiceAccountName        = serviceAccountName
	ExportTailscaleAuthorizationKey = tailscaleAuthorizationKey
)

func ExportSetHeadscaleGRPCClientOnHAR(r *HeadscaleAccessReconciler, c headscalev1.HeadscaleServiceClient) {
	r.headscaleGRPCClient = c
}

func ExportSetRestClientGetterFunc(r *HeadscaleAccessReconciler, f func(restClientGetter genericclioptions.RESTClientGetter, proxy string, headscaleAddress string) (client.Client, error)) {
	r.getHeadscaleClientFromRestClientGetter = f
}

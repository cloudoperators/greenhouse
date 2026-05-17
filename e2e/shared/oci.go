// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package shared

import (
	"context"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// CreateMirrorConfigMap creates or updates the OCI registry mirror ConfigMap in the given namespace
// with the standard mirror configuration used in e2e tests.
func CreateMirrorConfigMap(ctx context.Context, adminClient client.Client, namespace string) {
	mirrorCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oci-replication-config",
			Namespace: namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, adminClient, mirrorCM, func() error {
		mirrorCM.Data = map[string]string{
			"containerRegistryConfig": `primaryMirror: "registry:5000"
registryMirrors:
  ghcr.io:
    baseDomain: "registry:5000"
    subPath: "greenhouse-ghcr-io-mirror"`,
		}
		return nil
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error creating the mirror ConfigMap")
}

func SetupOCIMirroringForOrg(ctx context.Context, adminClient client.Client, orgName string) {
	CreateMirrorConfigMap(ctx, adminClient, orgName)
	patchOrgConfigMapRef(ctx, adminClient, orgName, "oci-replication-config")
}

func TeardownOCIMirroringForOrg(ctx context.Context, adminClient client.Client, orgName string) {
	patchOrgConfigMapRef(ctx, adminClient, orgName, "")
}

func patchOrgConfigMapRef(ctx context.Context, adminClient client.Client, orgName, ref string) {
	org := &greenhousev1alpha1.Organization{}
	Expect(adminClient.Get(ctx, client.ObjectKey{Name: orgName}, org)).To(Succeed())
	patch := client.MergeFrom(org.DeepCopy())
	org.Spec.ConfigMapRef = ref
	Expect(adminClient.Patch(ctx, org, patch)).To(Succeed())
}

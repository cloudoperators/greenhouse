// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases_test

import (
	"context"

	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/controller/plugindefinition/phases"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

var _ = Describe("CreateUpdateHelmChart", func() {
	DescribeTable("propagating the reconcile request to the HelmChart",
		func(annotations map[string]string, expectValue string) {
			pluginDef := &greenhousev1alpha1.PluginDefinition{
				ObjectMeta: metav1.ObjectMeta{Name: "test-plugin", Namespace: "test-org"},
			}
			pluginDef.SetAnnotations(annotations)
			pluginDef.Spec.HelmChart = &greenhousev1alpha1.HelmChartReference{
				Name:       "test-chart",
				Repository: "oci://registry.example.com/charts",
				Version:    "0.0.1",
			}

			scheme := runtime.NewScheme()
			Expect(greenhousev1alpha1.AddToScheme(scheme)).To(Succeed())
			Expect(sourcev1.AddToScheme(scheme)).To(Succeed())

			p := &phases.Phase{
				Client:        fake.NewClientBuilder().WithScheme(scheme).WithObjects(pluginDef).Build(),
				PluginDef:     pluginDef,
				NamespaceName: pluginDef.Namespace,
			}

			helmRepo := &sourcev1.HelmRepository{
				ObjectMeta: metav1.ObjectMeta{Name: "test-repo", Namespace: pluginDef.Namespace},
			}
			helmChart, err := p.CreateUpdateHelmChart(context.Background(), helmRepo)
			Expect(err).ToNot(HaveOccurred())
			Expect(helmChart.GetAnnotations()[fluxmeta.ReconcileRequestAnnotation]).To(Equal(expectValue))

			fetched := &sourcev1.HelmChart{}
			Expect(p.Client.Get(context.Background(), client.ObjectKeyFromObject(helmChart), fetched)).To(Succeed())
			Expect(fetched.GetAnnotations()[fluxmeta.ReconcileRequestAnnotation]).To(Equal(expectValue))
		},
		Entry("propagates the reconcile request annotation to the HelmChart",
			map[string]string{lifecycle.ReconcileAnnotation: "2026-08-25T13:34:13Z"}, "2026-08-25T13:34:13Z"),
		Entry("leaves the HelmChart untouched when no reconcile was requested",
			nil, ""),
	)
})

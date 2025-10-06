// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package scenarios

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	appsv1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/e2e/plugin/fixtures"
	"github.com/cloudoperators/greenhouse/e2e/shared"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const (
	preventDeletionAnnotation = "greenhouse.sap/prevent-deletion"
	nginxPluginOne            = "test-nginx-plugin-1"
	nginxPluginTwo            = "test-nginx-plugin-2"
	nginxPluginPreset         = "test-nginx-plugin-preset"
)

func createNginxPluginDefinition(ctx context.Context, adminClient client.Client, namespace string) *greenhousev1alpha1.ClusterPluginDefinition {
	By("Creating nginx plugin definition")
	testPluginDefinition := fixtures.PrepareNginxPluginDefinition(namespace)
	err := adminClient.Create(ctx, testPluginDefinition)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred())

	By("Checking the test plugin definition is ready")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginDefinition), testPluginDefinition)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(testPluginDefinition.Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras, Fields{
			"Type":   Equal(greenhousemetav1alpha1.ReadyCondition),
			"Status": Equal(metav1.ConditionTrue),
		})))
	}).Should(Succeed())
	return testPluginDefinition
}

func PluginControllerNginxByPlugin(ctx context.Context, adminClient, remoteClient client.Client, env *shared.TestEnv, remoteClusterName, teamName string) {
	testPluginDefinition := createNginxPluginDefinition(ctx, adminClient, env.TestNamespace)
	By("Creating the plugin")
	// Creating plugin with release name
	testPlugin := fixtures.PreparePlugin(nginxPluginOne, env.TestNamespace,
		test.WithClusterPluginDefinition(testPluginDefinition.Name),
		test.WithCluster(remoteClusterName),
		test.WithReleaseNamespace(env.TestNamespace),
		test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}),
		test.WithReleaseName(nginxPluginOne),
		test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
	)
	err := adminClient.Create(ctx, testPlugin)
	Expect(err).ToNot(HaveOccurred())

	By("Checking the plugin status is ready")
	plugin := &greenhousev1alpha1.Plugin{}
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, types.NamespacedName{Name: testPlugin.Name, Namespace: testPlugin.Namespace}, plugin)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(plugin.Status.HelmReleaseStatus).ToNot(BeNil())
		g.Expect(plugin.Status.HelmReleaseStatus.Status).To(BeEquivalentTo("deployed"))
	}).Should(Succeed())

	By("Checking deployment")
	ngInxDeployment := &appsv1.Deployment{}
	ngInxDeployment.SetName(nginxPluginOne)
	ngInxDeployment.SetNamespace(env.TestNamespace)
	Eventually(func(g Gomega) {
		err = remoteClient.Get(ctx, client.ObjectKeyFromObject(ngInxDeployment), ngInxDeployment)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(ngInxDeployment.Spec.Replicas).To(PointTo(Equal(int32(1))))
	}).Should(Succeed())

	By("Updating replicas")
	Eventually(func(g Gomega) {
		namespacedName := types.NamespacedName{Name: testPlugin.Name, Namespace: env.TestNamespace}
		err = adminClient.Get(ctx, namespacedName, testPlugin)
		g.Expect(err).NotTo(HaveOccurred())
		test.SetOptionValueForPlugin(testPlugin, "replicaCount", "2")
		err = adminClient.Update(ctx, testPlugin)
		g.Expect(err).NotTo(HaveOccurred())
	}).Should(Succeed())

	By("Check the diff status")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKey{Name: testPlugin.Name, Namespace: env.TestNamespace}, testPlugin)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(testPlugin.Status.HelmReleaseStatus).ToNot(BeNil())
		g.Expect(testPlugin.Status.HelmReleaseStatus.Diff).ToNot(BeEmpty(), "there should be a diff after updating the plugin")
	}).Should(Succeed())

	By("Check replicas in deployment")
	Eventually(func(g Gomega) {
		err = remoteClient.Get(ctx, client.ObjectKeyFromObject(ngInxDeployment), ngInxDeployment)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(ngInxDeployment.Spec.Replicas).To(PointTo(Equal(int32(2))))
	}).Should(Succeed())

	By("Deleting plugin")
	test.EventuallyDeleted(ctx, adminClient, testPlugin)

	By("Check, is deployment deleted")
	Eventually(func(g Gomega) {
		err = remoteClient.Get(ctx, client.ObjectKeyFromObject(ngInxDeployment), ngInxDeployment)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "the error should be NotFound")
	}).Should(Succeed(), "the deployment should be deleted")

	By("Deleting plugin definition")
	test.EventuallyDeleted(ctx, adminClient, testPluginDefinition)
}

func PluginControllerNginxByPreset(ctx context.Context, adminClient, remoteClient client.Client, env *shared.TestEnv, remoteClusterName, teamName string) {
	testPluginDefinition := createNginxPluginDefinition(ctx, adminClient, env.TestNamespace)

	By("Prepare the plugin")
	// Creating plugin with release name
	testPlugin := fixtures.PreparePlugin(nginxPluginTwo, env.TestNamespace,
		test.WithClusterPluginDefinition(testPluginDefinition.Name),
		test.WithReleaseNamespace(env.TestNamespace),
		test.WithPluginOptionValue("replicaCount", &apiextensionsv1.JSON{Raw: []byte("1")}),
		test.WithReleaseName(nginxPluginTwo),
		test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
	)

	By("Add labels to remote cluster")
	remoteCluster := &greenhousev1alpha1.Cluster{}
	err := adminClient.Get(ctx, client.ObjectKey{Name: remoteClusterName, Namespace: env.TestNamespace}, remoteCluster)
	Expect(err).ToNot(HaveOccurred())
	remoteCluster.Labels["app"] = "test-cluster"
	err = adminClient.Update(ctx, remoteCluster)
	Expect(err).ToNot(HaveOccurred())

	By("Creating the plugin preset")
	testPluginPreset := test.NewPluginPreset(nginxPluginPreset, env.TestNamespace,
		test.WithPluginPresetLabel(greenhouseapis.LabelKeyOwnedBy, teamName),
		test.WithPluginPresetPluginSpec(testPlugin.Spec),
		test.WithPluginPresetClusterSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": "test-cluster",
			},
		}),
	)
	err = adminClient.Create(ctx, testPluginPreset)
	Expect(client.IgnoreAlreadyExists(err)).ToNot(HaveOccurred(), "there should be no error creating the PluginPreset")

	By("Checking the plugin status is ready")
	Eventually(func(g Gomega) {
		pluginList := &greenhousev1alpha1.PluginList{}
		err = adminClient.List(
			ctx,
			pluginList,
			client.InNamespace(env.TestNamespace),
			client.MatchingLabels{
				greenhouseapis.LabelKeyPluginPreset: testPluginPreset.Name,
			},
		)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(len(pluginList.Items)).To(BeEquivalentTo(1))
	}).Should(Succeed())

	By("Check the replicas in deployment")
	ngInxDeployment := &appsv1.Deployment{}
	ngInxDeployment.SetName(nginxPluginTwo)
	ngInxDeployment.SetNamespace(env.TestNamespace)
	Eventually(func(g Gomega) {
		err = remoteClient.Get(ctx, client.ObjectKeyFromObject(ngInxDeployment), ngInxDeployment)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(ngInxDeployment.Spec.Replicas).To(PointTo(Equal(int32(1))))
	}).Should(Succeed())

	By("Updating plugin preset with cluster overview")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginPreset), testPluginPreset)
		g.Expect(err).ToNot(HaveOccurred(), "failed to get the PluginPreset")
		testPluginPreset.Spec.ClusterOptionOverrides = []greenhousev1alpha1.ClusterOptionOverride{
			{
				ClusterName: remoteClusterName,
				Overrides: []greenhousev1alpha1.PluginOptionValue{
					{
						Name:  "replicaCount",
						Value: &apiextensionsv1.JSON{Raw: []byte("2")},
					},
				},
			},
		}
		err = adminClient.Update(ctx, testPluginPreset)
		g.Expect(err).ToNot(HaveOccurred(), "failed to update the PluginPreset")
	}).Should(Succeed(), "there should be no error updating the PluginPreset")

	By("Check replicas in deployment")
	Eventually(func(g Gomega) {
		err = remoteClient.Get(ctx, client.ObjectKeyFromObject(ngInxDeployment), ngInxDeployment)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(ngInxDeployment.Spec.Replicas).To(PointTo(Equal(int32(2))))
	}).Should(Succeed())

	By("Updating plugin preset with cluster option override")
	err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginPreset), testPluginPreset)
	Expect(err).ToNot(HaveOccurred())
	testPluginPreset.Spec.ClusterOptionOverrides = []greenhousev1alpha1.ClusterOptionOverride{
		{
			ClusterName: remoteClusterName,
			Overrides: []greenhousev1alpha1.PluginOptionValue{
				{
					Name:  "replicaCount",
					Value: &apiextensionsv1.JSON{Raw: []byte("3")},
				},
			},
		},
	}
	err = adminClient.Update(ctx, testPluginPreset)
	Expect(err).ToNot(HaveOccurred(), "there should be no error updating the plugin preset with override")

	By("Ensure the plugin preset is updated")
	Eventually(func(g Gomega) {
		err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginPreset), testPluginPreset)
		Expect(err).ToNot(HaveOccurred())
		Expect(testPluginPreset.Spec.ClusterOptionOverrides).To(HaveLen(1))
		Expect(testPluginPreset.Spec.ClusterOptionOverrides[0].Overrides).To(HaveLen(1))
		Expect(testPluginPreset.Spec.ClusterOptionOverrides[0].Overrides[0].Name).To(BeEquivalentTo("replicaCount"))
		Expect(testPluginPreset.Spec.ClusterOptionOverrides[0].Overrides[0].Value).To(BeEquivalentTo(&apiextensionsv1.JSON{Raw: []byte("3")}))
	}).Should(Succeed())

	By("Checking the plugin status is ready")
	Eventually(func(g Gomega) {
		pluginList := &greenhousev1alpha1.PluginList{}
		err = adminClient.List(
			ctx,
			pluginList,
			client.InNamespace(env.TestNamespace),
			client.MatchingLabels{
				greenhouseapis.LabelKeyPluginPreset: testPluginPreset.Name,
			},
		)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(len(pluginList.Items)).To(BeEquivalentTo(1))
		g.Expect(pluginList.Items[0].Status.HelmReleaseStatus.Status).To(BeEquivalentTo("deployed"))
	}).Should(Succeed())

	By("Check replicas in deployment")
	Eventually(func(g Gomega) {
		err = remoteClient.Get(ctx, client.ObjectKeyFromObject(ngInxDeployment), ngInxDeployment)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(ngInxDeployment.Spec.Replicas).To(PointTo(Equal(int32(3))))
	}).Should(Succeed())

	By("Deleting the plugin preset")
	err = adminClient.Get(ctx, client.ObjectKeyFromObject(testPluginPreset), testPluginPreset)
	Expect(err).ToNot(HaveOccurred())
	// Remove prevent-deletion annotation before deleting plugin preset.
	_, err = clientutil.Patch(ctx, adminClient, testPluginPreset, func() error {
		delete(testPluginPreset.Annotations, preventDeletionAnnotation)
		return nil
	})
	Expect(err).ToNot(HaveOccurred(), "there should be no error removing the prevent-deletion annotation")
	test.EventuallyDeleted(ctx, adminClient, testPluginPreset)

	By("Check that the deployment is deleted")
	Eventually(func(g Gomega) {
		err = remoteClient.Get(ctx, client.ObjectKeyFromObject(ngInxDeployment), ngInxDeployment)
		g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "the error should be NotFound")
	}).Should(Succeed(), "deployment should be deleted")

	By("Deleting the plugin definition")
	test.EventuallyDeleted(ctx, adminClient, testPluginDefinition)
}

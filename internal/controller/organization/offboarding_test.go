// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization_test

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	dexapi "github.com/cloudoperators/greenhouse/internal/dex/api"
	"github.com/cloudoperators/greenhouse/internal/test"
)

const (
	foreignGroup   = "foreign.greenhouse.test"
	unrelatedGroup = "unrelated.greenhouse.test"
	holdFinalizer  = "greenhouse.sap/test-hold"
	orderWindow    = 2 * time.Second
)

var _ = Describe("Organization offboarding", Ordered, ContinueOnFailure, func() {
	var (
		setup *test.TestSetup
		org   *greenhousev1alpha1.Organization
	)

	BeforeAll(func() {
		createCRD(foreignGroup, "Widget", apiextensionsv1.NamespaceScoped)
		createCRD(foreignGroup, "Gadget", apiextensionsv1.ClusterScoped)
		createCRD(unrelatedGroup, "Widget", apiextensionsv1.NamespaceScoped)
	})

	BeforeEach(func() {
		setup = test.NewTestSetup(test.Ctx, test.K8sClient, test.TestNamespace)
		org = setup.CreateOrganization(test.Ctx, setup.Namespace(), test.WithMappedAdminIDPGroup("OFFBOARDING_IDP_GROUP"))
		Eventually(func(g Gomega) {
			g.Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(org), org)).To(Succeed())
			g.Expect(org.GetFinalizers()).ToNot(BeEmpty())
		}).Should(Succeed())
	})

	It("deletes the resources in order", func() {
		team := setup.CreateTeam(test.Ctx, "team", test.WithMappedIDPGroup("OFFBOARDING_TEAM_GROUP"))
		teamRole := setup.CreateTeamRole(test.Ctx, "role")
		cluster := setup.CreateCluster(test.Ctx, "cluster")
		pluginDefinition := setup.CreatePluginDefinition(test.Ctx, "definition")
		plugin := setup.CreatePlugin(test.Ctx, "plugin",
			test.WithPluginDefinition(pluginDefinition.Name),
			test.WithCluster(cluster.Name),
			test.WithReleaseName("release"),
		)
		preset := setup.CreatePluginPreset(test.Ctx, "preset",
			test.WithPluginPresetClusterSelector(metav1.LabelSelector{MatchLabels: map[string]string{"offboarding": "true"}}),
			test.WithPluginPresetPluginSpec(greenhousev1alpha1.PluginSpec{
				PluginDefinitionRef: greenhousev1alpha1.PluginDefinitionReference{
					Name: pluginDefinition.Name,
					Kind: greenhousev1alpha1.PluginDefinitionKind,
				},
				ReleaseName:      "preset-release",
				ReleaseNamespace: org.Name,
			}),
		)
		binding := setup.CreateTeamRoleBinding(test.Ctx, "binding",
			test.WithTeamRoleRef(teamRole.Name),
			test.WithTeamRefs(team.Name),
			test.WithClusterName(cluster.Name),
		)
		widget := createObject(foreignGroup, "Widget", org.Name, "widget")

		ordered := []client.Object{widget, binding, cluster, preset, plugin}
		for _, obj := range ordered {
			setFinalizer(obj, controllerutil.AddFinalizer)
		}

		Expect(test.K8sClient.Delete(test.Ctx, org)).To(Succeed())

		for i, obj := range ordered {
			By("waiting for the deletion of " + obj.GetName())
			Eventually(func(g Gomega) {
				expectTerminating(g, obj)
			}).Should(Succeed())
			Consistently(func(g Gomega) {
				for _, later := range ordered[i+1:] {
					g.Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(later), later)).To(Succeed())
					g.Expect(later.GetDeletionTimestamp()).To(BeNil(), later.GetName()+" must wait for "+obj.GetName())
				}
			}).WithTimeout(orderWindow).Should(Succeed())
			setFinalizer(obj, controllerutil.RemoveFinalizer)
		}

		expectGone(org)
	})

	It("labels the namespace and leaves it behind without the label or an owner reference", func() {
		namespace := &corev1.Namespace{}
		Eventually(func(g Gomega) {
			g.Expect(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: org.Name}, namespace)).To(Succeed())
			g.Expect(namespace.GetLabels()).To(HaveKeyWithValue(greenhouseapis.LabelKeyOrganization, org.Name))
		}).Should(Succeed())
		Expect(namespace.GetOwnerReferences()).To(BeEmpty())

		Eventually(func(g Gomega) {
			g.Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(namespace), namespace)).To(Succeed())
			g.Expect(controllerutil.SetControllerReference(org, namespace, test.K8sClient.Scheme())).To(Succeed())
			g.Expect(test.K8sClient.Update(test.Ctx, namespace)).To(Succeed())
		}).Should(Succeed())

		test.EventuallyDeleted(test.Ctx, test.K8sClient, org)

		Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(namespace), namespace)).To(Succeed())
		Expect(namespace.GetOwnerReferences()).To(BeEmpty())
		Expect(namespace.GetLabels()).NotTo(HaveKey(greenhouseapis.LabelKeyOrganization))
		Expect(namespace.GetDeletionTimestamp()).To(BeNil())
	})

	It("deletes the Secret of a Cluster", func() {
		cluster := setup.CreateCluster(test.Ctx, "cluster")
		secret := setup.CreateSecret(test.Ctx, cluster.Name)
		setFinalizer(cluster, controllerutil.AddFinalizer)

		Expect(test.K8sClient.Delete(test.Ctx, org)).To(Succeed())

		expectGone(secret)
		setFinalizer(cluster, controllerutil.RemoveFinalizer)
		expectGone(org)
	})

	It("leaves resources outside of the configured groups and the namespace alone", func() {
		gadget := createObject(foreignGroup, "Gadget", "", org.Name)
		unrelated := createObject(unrelatedGroup, "Widget", org.Name, "unrelated")
		otherNamespace := createObject(foreignGroup, "Widget", test.TestNamespace, org.Name)

		test.EventuallyDeleted(test.Ctx, test.K8sClient, org)

		for _, obj := range []client.Object{gadget, unrelated, otherNamespace} {
			Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(obj), obj)).To(Succeed())
			Expect(obj.GetDeletionTimestamp()).To(BeNil(), obj.GetName()+" must not be deleted")
		}
	})

	It("deletes the service proxy oauth client", func() {
		team := setup.CreateTeam(test.Ctx, "team", test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
		setup.CreateDefaultOrgWithOIDCSecret(test.Ctx, team.Name)
		Expect(test.K8sClient.Create(test.Ctx, test.NewPluginDefinition(test.Ctx, "service-proxy", org.Name))).To(Succeed())
		secret := setup.CreateOrgOIDCSecret(test.Ctx, org.Name, team.Name)
		setup.UpdateOrganization(test.Ctx, org.Name, test.WithOIDCConfig(test.OIDCIssuer, secret.Name, test.OIDCClientIDKey, test.OIDCClientSecretKey))

		proxyClient := "oauth2-proxy--" + org.Name
		Eventually(oauthClientIDs).Should(ContainElement(proxyClient))

		test.EventuallyDeleted(test.Ctx, test.K8sClient, org)

		Eventually(oauthClientIDs).ShouldNot(ContainElement(BeElementOf(proxyClient, org.Name)))
	})
})

func createCRD(group, kind string, scope apiextensionsv1.ResourceScope) {
	GinkgoHelper()
	singular := strings.ToLower(kind)
	schemaProps := &apiextensionsv1.CustomResourceValidation{
		OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{Type: "object", XPreserveUnknownFields: new(true)},
	}
	crd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: singular + "s." + group},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: group,
			Scope: scope,
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Kind:     kind,
				ListKind: kind + "List",
				Plural:   singular + "s",
				Singular: singular,
			},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{Name: "v1alpha1", Served: false, Storage: true, Schema: schemaProps},
				{Name: "v1", Served: true, Storage: false, Schema: schemaProps},
			},
		},
	}
	Expect(test.K8sClient.Create(test.Ctx, crd)).To(Succeed())
	Eventually(func(g Gomega) {
		g.Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(crd), crd)).To(Succeed())
		established := false
		for _, condition := range crd.Status.Conditions {
			if condition.Type == apiextensionsv1.Established {
				established = condition.Status == apiextensionsv1.ConditionTrue
			}
		}
		g.Expect(established).To(BeTrue())
	}).Should(Succeed())
}

func createObject(group, kind, namespace, name string) *unstructured.Unstructured {
	GinkgoHelper()
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: group, Version: "v1", Kind: kind})
	obj.SetNamespace(namespace)
	obj.SetName(name)
	Eventually(func() error {
		return test.K8sClient.Create(test.Ctx, obj)
	}).Should(Succeed())
	return obj
}

func setFinalizer(obj client.Object, mutate func(client.Object, string) bool) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		g.Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(obj), obj)).To(Succeed())
		mutate(obj, holdFinalizer)
		g.Expect(test.K8sClient.Update(test.Ctx, obj)).To(Succeed())
	}).Should(Succeed())
}

func expectTerminating(g Gomega, obj client.Object) {
	g.Expect(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(obj), obj)).To(Succeed())
	g.Expect(obj.GetDeletionTimestamp()).ToNot(BeNil(), obj.GetName()+" should be terminating")
}

func expectGone(obj client.Object) {
	GinkgoHelper()
	Eventually(func() bool {
		return apierrors.IsNotFound(test.K8sClient.Get(test.Ctx, client.ObjectKeyFromObject(obj), obj))
	}).Should(BeTrue(), obj.GetName()+" should be gone")
}

func oauthClientIDs() []string {
	GinkgoHelper()
	clients := &dexapi.OAuth2ClientList{}
	Expect(test.K8sClient.List(test.Ctx, clients)).To(Succeed())
	ids := make([]string, 0, len(clients.Items))
	for _, oauthClient := range clients.Items {
		ids = append(ids, oauthClient.ID)
	}
	return ids
}

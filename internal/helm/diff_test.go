// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/helm"
	"github.com/cloudoperators/greenhouse/internal/test"
)

var (
	secret           *corev1.Secret
	secretName       = "test-secret"
	stringSecretName = "test-string-secret"
	stringSecret     *corev1.Secret
)

var _ = Describe("ensure helm diff against the release manifest works as expected", func() {
	var (
		pluginDefinitionUT *greenhousev1alpha1.ClusterPluginDefinition
		pluginUT           *greenhousev1alpha1.Plugin
	)

	BeforeEach(func() {
		pluginDefinitionUT = test.NewClusterPluginDefinition(test.Ctx, "test-plugindefinition",
			test.WithHelmChart(&greenhousev1alpha1.HelmChartReference{
				Name:    "./../test/fixtures/myChart",
				Version: "1.0.0",
			}),
		)

		pluginUT = test.NewPlugin(test.Ctx, "test-plugin", namespace,
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, "test-team-1"),
			test.WithClusterPluginDefinition("test-plugindefinition"),
			test.WithPluginOptionValue("enabled", test.MustReturnJSONFor(true)),
			test.WithReleaseNamespace(namespace),
		)

		// install the chart
		r, err := helm.ExportInstallHelmRelease(test.Ctx, test.K8sClient, test.RestClientGetter, pluginDefinitionUT.Spec, pluginUT, false)
		Expect(err).NotTo(HaveOccurred(), "there should be no error installing the helm release")
		Expect(r).NotTo(BeNil(), "the release should not be nil")
	})

	AfterEach(func() {
		_, err := helm.UninstallHelmRelease(test.Ctx, test.RestClientGetter, pluginUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error uninstalling the helm release")
	})

	It("should no diff or drift if nothing changes", func() {
		By("templating the Helm Chart from the Plugin")
		templateUT, err := helm.TemplateHelmChartFromPlugin(test.Ctx, test.K8sClient, test.RestClientGetter, pluginDefinitionUT.Spec, pluginUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error templating the helm chart")

		By("retrieving the Release for the Plugin")
		releaseUT, err := helm.GetReleaseForHelmChartFromPlugin(test.Ctx, test.RestClientGetter, pluginUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error getting the release for the helm chart")

		By("diffing the manifest against the helm release")
		diff, err := helm.ExportDiffAgainstRelease(test.RestClientGetter, namespace, templateUT, releaseUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error diffing the manifest against the helm release")
		Expect(diff).To(BeEmpty(), "the diff should be empty")

		By("diffing the manifest against the live objects")
		diff, err = helm.ExportDiffAgainstLiveObjects(test.RestClientGetter, namespace, templateUT.Manifest)
		Expect(err).NotTo(HaveOccurred(), "there should be no error diffing the manifest against the helm release")
		Expect(diff).To(BeEmpty(), "the diff should be empty")
	})

	It("should show a diff and a drift if the Plugin is changed", func() {
		By("changing the Plugin's PluginOptionValues")
		pluginUT.Spec.OptionValues = []greenhousev1alpha1.PluginOptionValue{
			{
				Name:  "imageTag",
				Value: test.MustReturnJSONFor("3.19"),
			},
		}

		By("templating the Helm Chart from the Plugin")
		templateUT, err := helm.TemplateHelmChartFromPlugin(test.Ctx, test.K8sClient, test.RestClientGetter, pluginDefinitionUT.Spec, pluginUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error templating the helm chart")

		By("retrieving the Release for the Plugin")
		releaseUT, err := helm.GetReleaseForHelmChartFromPlugin(test.Ctx, test.RestClientGetter, pluginUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error getting the release for the helm chart")

		By("diffing the manifest against the helm release")
		diff, err := helm.ExportDiffAgainstRelease(test.RestClientGetter, namespace, templateUT, releaseUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error diffing the manifest against the helm release")
		Expect(diff).To(ContainSubstring("3.19"), "the diff should not be empty")

		By("diffing the manifest against the live objects")
		diff, err = helm.ExportDiffAgainstLiveObjects(test.RestClientGetter, namespace, templateUT.Manifest)
		Expect(err).NotTo(HaveOccurred(), "there should be no error diffing the manifest against the helm release")
		Expect(diff).To(ContainSubstring("3.19"), "the diff should not be empty")
	})

	It("should show a diff if a template was disabled", func() {
		By("changing the Plugin's PluginOptionValues")
		pluginUT.Spec.OptionValues = []greenhousev1alpha1.PluginOptionValue{
			{
				Name:  "enabled",
				Value: test.MustReturnJSONFor(false),
			},
		}

		By("templating the Helm Chart from the Plugin")
		manifestUT, err := helm.TemplateHelmChartFromPlugin(test.Ctx, test.K8sClient, test.RestClientGetter, pluginDefinitionUT.Spec, pluginUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error templating the helm chart")

		By("retrieving the Release for the Plugin")
		releaseUT, err := helm.GetReleaseForHelmChartFromPlugin(test.Ctx, test.RestClientGetter, pluginUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error getting the release for the helm chart")

		By("diffing the manifest against the helm release")
		diff, err := helm.ExportDiffAgainstRelease(test.RestClientGetter, namespace, manifestUT, releaseUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error diffing the manifest against the helm release")
		Expect(diff).To(ContainSubstring("Pod/alpine-flag"), "the diff should contain the disabled pod")
	})

	It("should show no diff but a drift if the remote object was changed", func() {
		By("changing the Image of the pod")
		podUT := &corev1.Pod{}
		Eventually(func() error {
			return test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: "alpine", Namespace: namespace}, podUT)
		}).Should(Succeed(), "the pod should be retrieved")

		podUT.Spec.Containers[0].Image = "alpine:3.19"
		Expect(test.K8sClient.Update(test.Ctx, podUT)).To(Succeed(), "the pod should be updated")

		By("templating the Helm Chart from the Plugin")
		templateUT, err := helm.TemplateHelmChartFromPlugin(test.Ctx, test.K8sClient, test.RestClientGetter, pluginDefinitionUT.Spec, pluginUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error templating the helm chart")

		By("retrieving the Release for the Plugin")
		releaseUT, err := helm.GetReleaseForHelmChartFromPlugin(test.Ctx, test.RestClientGetter, pluginUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error getting the release for the helm chart")

		By("diffing the manifest against the helm release")
		diff, err := helm.ExportDiffAgainstRelease(test.RestClientGetter, namespace, templateUT, releaseUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error diffing the manifest against the helm release")
		Expect(diff).To(BeEmpty(), "the diff should be empty")

		By("diffing the manifest against the live objects")
		diff, err = helm.ExportDiffAgainstLiveObjects(test.RestClientGetter, namespace, templateUT.Manifest)
		Expect(err).NotTo(HaveOccurred(), "there should be no error diffing the manifest against the helm release")
		Expect(diff).To(ContainSubstring("3.18"), "the diff should not be empty")
	})
})

var _ = Describe("ensure helm with hooks diff against the release manifest works as expected", Ordered, func() {
	var (
		pluginDefinitionUT *greenhousev1alpha1.ClusterPluginDefinition
		pluginUT           *greenhousev1alpha1.Plugin
	)

	BeforeEach(func() {
		pluginDefinitionUT = test.NewClusterPluginDefinition(test.Ctx, "test-plugindefinition",
			test.WithHelmChart(&greenhousev1alpha1.HelmChartReference{
				Name:    "./../test/fixtures/testHook",
				Version: "1.0.0",
			}))
		pluginUT = test.NewPlugin(test.Ctx, "test-plugin", namespace,
			test.WithPluginLabel(greenhouseapis.LabelKeyOwnedBy, "test-team-1"),
			test.WithClusterPluginDefinition("test-plugindefinition"),
			test.WithPluginOptionValue("hook_enabled", test.MustReturnJSONFor(false)),
			test.WithReleaseNamespace(namespace),
		)

		By("install the chart")
		r, err := helm.ExportInstallHelmRelease(test.Ctx, test.K8sClient, test.RestClientGetter, pluginDefinitionUT.Spec, pluginUT, false)
		Expect(err).NotTo(HaveOccurred(), "there should be no error installing the helm release")
		Expect(r).NotTo(BeNil(), "the release should not be nil")
	})

	AfterEach(func() {
		_, err := helm.UninstallHelmRelease(test.Ctx, test.RestClientGetter, pluginUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error uninstalling the helm release")
	})

	It("should show a diff if a plugin was installed", func() {
		By("templating the Helm Chart from the Plugin")
		manifestUT, err := helm.TemplateHelmChartFromPlugin(test.Ctx, test.K8sClient, test.RestClientGetter, pluginDefinitionUT.Spec, pluginUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error templating the helm chart")

		By("retrieving the Release for the Plugin")
		releaseUT, err := helm.GetReleaseForHelmChartFromPlugin(test.Ctx, test.RestClientGetter, pluginUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error getting the release for the helm chart")

		By("diffing the manifest against the helm release")
		diff, err := helm.ExportDiffAgainstRelease(test.RestClientGetter, namespace, manifestUT, releaseUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error diffing the manifest against the helm release")
		Expect(diff).To(BeEmpty(), "the diff should not be empty")

	})

	It("should show a diff if a plugin was installed", func() {
		By("changing the Plugin's PluginOptionValues")
		pluginUT.Spec.OptionValues = []greenhousev1alpha1.PluginOptionValue{
			{
				Name:  "hook_enabled",
				Value: test.MustReturnJSONFor(true),
			},
		}

		By("templating the Helm Chart from the Plugin")
		manifestUT, err := helm.TemplateHelmChartFromPlugin(test.Ctx, test.K8sClient, test.RestClientGetter, pluginDefinitionUT.Spec, pluginUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error templating the helm chart")

		By("retrieving the Release for the Plugin")
		releaseUT, err := helm.GetReleaseForHelmChartFromPlugin(test.Ctx, test.RestClientGetter, pluginUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error getting the release for the helm chart")

		By("diffing the manifest against the helm release")
		diff, err := helm.ExportDiffAgainstRelease(test.RestClientGetter, namespace, manifestUT, releaseUT)
		Expect(err).NotTo(HaveOccurred(), "there should be no error diffing the manifest against the helm release")
		Expect(diff).NotTo(BeEmpty(), "the diff should not be empty")
	})
})

var _ = Describe("ensure errors with Manifests are handled correctly", func() {
	It("should not return an error if the CRD does not exist", func() {
		manifest := `
      apiVersion: openstack.stable.sap.cc/v1
      kind: OpenstackSeed
      metadata:
        name: test
        namespace: test-org
      spec:
        domains:
        - name: test
          users:
            - name: test
              password: test
              role-assignments:
                 - name: test`

		diffs, err := helm.ExportDiffAgainstLiveObjects(test.RestClientGetter, namespace, manifest)
		Expect(err).NotTo(HaveOccurred(), "there should be an error diffing the helm release")
		Expect(diffs).To(BeEmpty(), "the diff should be empty")
	})
	It("should return an error because the manifest is malformatted", func() {
		manifest := `
      apiVersion: v1
      kind: Secret
      metadata:
        name: test-secret
        namespace: test-org
      type: Opaque
      data:
      test: dXBkYXRlZAo=
        cert: Y2VydGlmaWNhdGUgZGF0YQ==`

		diffs, err := helm.ExportDiffAgainstLiveObjects(test.RestClientGetter, namespace, manifest)
		Expect(err).To(HaveOccurred(), "there should be an error diffing the helm release")
		Expect(diffs).To(BeEmpty(), "the diff should be empty")
	})
})

var _ = Describe("Ensure helm diff does not leak secrets", Ordered, func() {
	var teamForDiffLeak *greenhousev1alpha1.Team
	BeforeAll(func() {
		By("creating a test Team")
		teamForDiffLeak = test.NewTeam(test.Ctx, "test-diff-leak-team", namespace, test.WithTeamLabel(greenhouseapis.LabelKeySupportGroup, "true"))
		Expect(test.K8sClient.Create(test.Ctx, teamForDiffLeak)).To(Succeed(), "there should be no error creating a Team")
		By("setting up the secrets for diffing")
		secret = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
				Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "test-diff-leak-team"},
			},
			Data: map[string][]byte{
				"test": []byte("test-value"),       // test-value => dGVzdC12YWx1ZQ==
				"cert": []byte("certificate-data"), // certificate data => Y2VydGlmaWNhdGUtZGF0YQ==
			},
		}
		data, err := runtime.Encode(unstructured.UnstructuredJSONScheme, secret)
		Expect(err).NotTo(HaveOccurred(), "there should be no error encoding the object")
		patch := client.RawPatch(types.ApplyPatchType, data)
		err = test.K8sClient.Patch(test.Ctx, secret, patch, &client.PatchOptions{FieldManager: helm.ExportGreenhouseFieldManager, Force: ptr.To(true)})
		Expect(err).NotTo(HaveOccurred(), "there should be no error creating the secret")
		secretID := types.NamespacedName{Name: secretName, Namespace: namespace}
		Eventually(func() error {
			return test.K8sClient.Get(test.Ctx, secretID, secret)
		}).Should(Succeed(), "the secret should be created")

		stringSecret = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      stringSecretName,
				Namespace: namespace,
				Labels:    map[string]string{greenhouseapis.LabelKeyOwnedBy: "test-diff-leak-team"},
			},
			StringData: map[string]string{
				"test": "test-value",
				"cert": "certificate-data",
			},
		}
		data, err = runtime.Encode(unstructured.UnstructuredJSONScheme, stringSecret)
		Expect(err).NotTo(HaveOccurred(), "there should be no error encoding the object")
		patch = client.RawPatch(types.ApplyPatchType, data)
		err = test.K8sClient.Patch(test.Ctx, stringSecret, patch, &client.PatchOptions{FieldManager: helm.ExportGreenhouseFieldManager, Force: ptr.To(true)})
		Expect(err).NotTo(HaveOccurred(), "there should be no error creating the secret")
		secretID = types.NamespacedName{Name: stringSecretName, Namespace: namespace}
		Eventually(func() error {
			return test.K8sClient.Get(test.Ctx, secretID, stringSecret)
		}).Should(Succeed(), "the stringData secret should be created")
	})
	AfterAll(func() {
		test.EventuallyDeleted(test.Ctx, test.K8sClient, teamForDiffLeak)
	})
	When("a secret is changed", func() {
		It("should redact the original and changed values under data", func() {
			manifest := `
      apiVersion: v1
      kind: Secret
      metadata:
        name: test-secret
        namespace: test-org
        labels: {
          greenhouse.sap/owned-by: test-diff-leak-team
        }
      type: Opaque
      data:
        test: bmV3LXZhbHVlCg==
        cert: Y2VydGlmaWNhdGUtZGF0YQ==`

			diffs, err := helm.ExportDiffAgainstLiveObjects(test.RestClientGetter, namespace, manifest)
			Expect(err).NotTo(HaveOccurred(), "there should be no error diffing the helm release")
			Expect(diffs).NotTo(BeEmpty(), "the diff should not be empty")
			Expect(diffs).NotTo(ContainSubstring("dGVzdC12YWx1ZQ=="), "the diff should not contain the original value for test")
			Expect(diffs).NotTo(ContainSubstring("bmV3LXZhbHVlCg=="), "the diff should not contain the modified value for test")
			Expect(diffs).NotTo(ContainSubstring("Y2VydGlmaWNhdGUtZGF0YQ=="), "the diff should not contain the original value for cert data")
		})
		It("should redact the removed value under data", func() {
			manifest := `
      apiVersion: v1
      kind: Secret
      metadata:
        name: test-data-secret
        namespace: test-org
        labels: {
          greenhouse.sap/owned-by: test-diff-leak-team
        }
      type: Opaque
      data:
        test: dGVzdC12YWx1ZQ==`

			diffs, err := helm.ExportDiffAgainstLiveObjects(test.RestClientGetter, namespace, manifest)
			Expect(err).NotTo(HaveOccurred(), "there should be no error diffing the helm release")
			Expect(diffs).NotTo(BeEmpty(), "the diff should not be empty")
			Expect(diffs).NotTo(ContainSubstring("dGVzdC12YWx1ZQ=="), "the diff should not contain the original value for test")
			Expect(diffs).NotTo(ContainSubstring("Y2VydGlmaWNhdGUtZGF0YQ=="), "the diff should not contain the original value for cert data")
		})
		It("should redact the newly added values under data", func() {
			manifest := `
      apiVersion: v1
      kind: Secret
      metadata:
        name: test-secret
        namespace: test-org
        labels: {
          greenhouse.sap/owned-by: test-diff-leak-team
        }
      type: Opaque
      data:
        test: dGVzdC12YWx1ZQ==
        cert: Y2VydGlmaWNhdGUgZGF0YQo=
        new: bmV3LXZhbHVlCg==`

			diffs, err := helm.ExportDiffAgainstLiveObjects(test.RestClientGetter, namespace, manifest)
			Expect(err).NotTo(HaveOccurred(), "there should be no error diffing the helm release")
			Expect(diffs).NotTo(BeEmpty(), "the diff should not be empty")
			Expect(diffs).NotTo(ContainSubstring("dGVzdC12YWx1ZQ=="), "the diff should not contain the original value for test")
			Expect(diffs).NotTo(ContainSubstring("Y2VydGlmaWNhdGUtZGF0YQ=="), "the diff should not contain the original value for cert data")
			Expect(diffs).NotTo(ContainSubstring("bmV3LXZhbHVlCg=="), "the diff should not contain the original value for new")
		})
		It("should redact the original and changed values under stringData", func() {
			manifest := `
      apiVersion: v1
      kind: Secret
      metadata:
        name: test-string-secret
        namespace: test-org
        labels: {
          greenhouse.sap/owned-by: test-diff-leak-team
        }
      type: Opaque
      stringData:
        test: modified
        cert: certificate data`

			diffs, err := helm.ExportDiffAgainstLiveObjects(test.RestClientGetter, namespace, manifest)
			Expect(err).NotTo(HaveOccurred(), "there should be no error diffing the helm release")
			Expect(diffs).NotTo(BeEmpty(), "the diff should not be empty")
			Expect(diffs).NotTo(ContainSubstring("test-value"), "the diff should not contain the original value for test")
			Expect(diffs).NotTo(ContainSubstring("modified"), "the diff should not contain the modified value for test")
			Expect(diffs).NotTo(ContainSubstring("certificate-data"), "the diff should not contain the original value for cert")
		})
		// Secrets with stringData seem to behave differently when removing a value
		// https://github.com/kubernetes/kubernetes/issues/118519
		// FIt("should redact the removed value under stringData", func() {
		// 	manifest := `
		//   apiVersion: v1
		//   kind: Secret
		//   metadata:
		//     name: test-string-secret
		//     namespace: test-org
		//   type: Opaque
		//   stringData:
		//     test: test-value`

		// 	diffs, err := helm.ExportDiffHelmRelease(test.RestClientGetter, namespace, manifest)
		// 	Expect(err).NotTo(HaveOccurred(), "there should be no error diffing the helm release")
		// 	Expect(diffs).NotTo(BeEmpty(), "the diff should not be empty")
		// 	Expect(diffs).NotTo(ContainSubstring("test-value"), "the diff should not contain the original value for test")
		// 	Expect(diffs).NotTo(ContainSubstring("certificate-data"), "the diff should not contain the original value for cert")
		// })
		It("should redact the new value under stringData", func() {
			manifest := `
      apiVersion: v1
      kind: Secret
      metadata:
        name: test-string-secret
        namespace: test-org
        labels: {
          greenhouse.sap/owned-by: test-diff-leak-team
        }
      type: Opaque
      stringData:
        test: test-value
        cert: certificate data
        new: new-value`

			diffs, err := helm.ExportDiffAgainstLiveObjects(test.RestClientGetter, namespace, manifest)
			Expect(err).NotTo(HaveOccurred(), "there should be no error diffing the helm release")
			Expect(diffs).NotTo(BeEmpty(), "the diff should not be empty")
			Expect(diffs).NotTo(ContainSubstring("test-value"), "the diff should not contain the original value for test")
			Expect(diffs).NotTo(ContainSubstring("certificate-data"), "the diff should not contain the original value for cert")
			Expect(diffs).NotTo(ContainSubstring("new-value"), "the diff should not contain the value for new")
		})
		It("should redact the secret also if there is no live object", func() {
			manifest := `
      apiVersion: v1
      kind: Secret
      metadata:
        name: new-secret
        namespace: test-org
        labels: {
          greenhouse.sap/owned-by: test-diff-leak-team
        }
      type: Opaque
      data:
        test: dGVzdC12YWx1ZQ==
        cert: Y2VydGlmaWNhdGUgZGF0YQo=`

			diffs, err := helm.ExportDiffAgainstLiveObjects(test.RestClientGetter, namespace, manifest)
			Expect(err).NotTo(HaveOccurred(), "there should be no error diffing the helm release")
			Expect(diffs).NotTo(BeEmpty(), "the diff should not be empty")
			Expect(diffs).NotTo(ContainSubstring("dGVzdC12YWx1ZQ=="), "the diff should not contain the original value for test")
			Expect(diffs).NotTo(ContainSubstring("Y2VydGlmaWNhdGUtZGF0YQ=="), "the diff should not contain the original value for cert data")
			Expect(diffs).NotTo(ContainSubstring("bmV3LXZhbHVlCg=="), "the diff should not contain the original value for new")
		})
		It("should not error if the secret data is nil", func() {
			manifest := `
      apiVersion: v1
      kind: Secret
      metadata:
        name: test-secret
        namespace: test-org
        labels: {
          greenhouse.sap/owned-by: test-diff-leak-team
        }
      type: Opaque
      data:`

			helmRelease := &release.Release{Manifest: manifest}

			diffs, err := helm.ExportDiffAgainstRelease(test.RestClientGetter, namespace, helmRelease, helmRelease)
			Expect(err).NotTo(HaveOccurred(), "there should be no error diffing the helm release")
			Expect(diffs).To(BeEmpty(), "the diff should be empty")
			Expect(diffs).NotTo(ContainSubstring("dGVzdC12YWx1ZQ=="), "the diff should not contain the original value for test")
			Expect(diffs).NotTo(ContainSubstring("Y2VydGlmaWNhdGUtZGF0YQ=="), "the diff should not contain the original value for cert data")
			Expect(diffs).NotTo(ContainSubstring("bmV3LXZhbHVlCg=="), "the diff should not contain the original value for new")
		})
	})
})

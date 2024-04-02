// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package controllers_test

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/controllers/cluster"
	"github.com/cloudoperators/greenhouse/pkg/controllers/fixtures"
	"github.com/cloudoperators/greenhouse/pkg/controllers/plugin"
	"github.com/cloudoperators/greenhouse/pkg/controllers/team"
	"github.com/cloudoperators/greenhouse/pkg/controllers/teammembership"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var (
	otherNamespace = "other-namespace"
	dummyCRDName   = "dummies.greenhouse.sap"
)

var testCluster = &greenhousev1alpha1.Cluster{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test-cluster",
		Namespace: test.TestNamespace,
	},
	TypeMeta: metav1.TypeMeta{
		Kind:       "Cluster",
		APIVersion: greenhousev1alpha1.GroupVersion.String(),
	},
	Spec: greenhousev1alpha1.ClusterSpec{
		AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
	},
}

var otherTestCluster = &greenhousev1alpha1.Cluster{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "other-test-cluster",
		Namespace: test.TestNamespace,
	},
	TypeMeta: metav1.TypeMeta{
		Kind:       "Cluster",
		APIVersion: greenhousev1alpha1.GroupVersion.String(),
	},
	Spec: greenhousev1alpha1.ClusterSpec{
		AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
	},
}

var dummy = &fixtures.Dummy{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "dummy",
		Namespace: test.TestNamespace,
	},
	TypeMeta: metav1.TypeMeta{
		Kind:       "Dummy",
		APIVersion: greenhousev1alpha1.GroupVersion.String(),
	},
	Spec: fixtures.DummySpec{
		Description: "test description",
		Property:    "test property",
	},
}

var _ = Describe("Propagation reconciler", Ordered, func() {
	BeforeAll(func() {
		// create namespaces
		//local
		err := test.K8sClient.Create(test.Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: otherNamespace}})
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the 2nd namespace in local cluster")
		// remote
		err = remoteClient.Create(test.Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: test.TestNamespace}})
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the test namespace in remote cluster")
		err = remoteClient.Create(test.Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: otherNamespace}})
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the 2nd namespace in remote cluster")
		// other remote
		err = otherRemoteClient.Create(test.Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: test.TestNamespace}})
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the test namespace in remote cluster")
		err = otherRemoteClient.Create(test.Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: otherNamespace}})
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the 2nd namespace in remote cluster")

		// create secrets
		err = test.K8sClient.Create(test.Ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testCluster.Name,
				Namespace: test.TestNamespace,
			},
			Type: greenhouseapis.SecretTypeKubeConfig,
			Data: map[string][]byte{
				greenhouseapis.KubeConfigKey: remoteKubeConfig,
			},
		})
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the first secret in the local cluster")
		err = test.K8sClient.Create(test.Ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      otherTestCluster.Name,
				Namespace: test.TestNamespace,
			},
			Type: greenhouseapis.SecretTypeKubeConfig,
			Data: map[string][]byte{
				greenhouseapis.KubeConfigKey: otherRemoteKubeConfig,
			},
		})
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the second secret in the local cluster")

		// create clusters
		err = test.K8sClient.Create(test.Ctx, testCluster)
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the first cluster resource in the local cluster")
		err = test.K8sClient.Create(test.Ctx, otherTestCluster)
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the second cluster resource in the local cluster")

		//create dummy crd
		dummyCRD, err := readCRDFromFile("./fixtures/dummy_crd.yaml")
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error reading the dummy crd file")
		err = test.K8sClient.Create(test.Ctx, dummyCRD)
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the dummy crd in the local cluster")
		err = envtest.WaitForCRDs(test.Cfg, []*apiextensionsv1.CustomResourceDefinition{dummyCRD}, envtest.CRDInstallOptions{MaxTime: 30 * time.Second, PollInterval: 1 * time.Second})
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error waiting for the dummy crd to be installed in the local cluster")

		// create dummy resource
		err = test.K8sClient.Create(test.Ctx, dummy)
		Expect(err).ShouldNot(HaveOccurred(), "there should be no error creating the dummy resource in the local cluster")
	})

	DescribeTable("should correctly strip objects", func(obj client.Object, r ObjectStripper, expErr bool, errSubString string) {

		strippedObj, err := r.StripObject(obj)
		switch expErr {
		case false:
			Expect(err).ShouldNot(HaveOccurred(), "there should be no error stripping the object")
			expectObjectsToMatch(strippedObj, obj)
		default:
			Expect(err).Should(HaveOccurred(), "there should be an error stripping the object")
			Expect(err.Error()).To(ContainSubstring(errSubString), "error message should contain the expected substring")
		}

	},
		Entry("cluster", &greenhousev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-cluster",
				Namespace:   test.TestNamespace,
				Labels:      map[string]string{"test-label": "test-value"},
				Annotations: map[string]string{"test-annotation": "test-value"},
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Cluster",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			Spec: greenhousev1alpha1.ClusterSpec{
				AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
			},
		}, &cluster.ClusterPropagationReconciler{}, false, ""),
		Entry("pluginDefinition", &greenhousev1alpha1.PluginDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-plugindefinition",
				Namespace:   test.TestNamespace,
				Labels:      map[string]string{"test-label": "test-value"},
				Annotations: map[string]string{"test-annotation": "test-value"},
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "PluginDefinition",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			Spec: greenhousev1alpha1.PluginDefinitionSpec{
				Description: "test description",
				Version:     "v1.1.1",
				HelmChart: &greenhousev1alpha1.HelmChartReference{
					Name:       "./../../test/fixtures/myChart",
					Repository: "dummy",
					Version:    "1.0.0",
				},
			},
		}, &plugin.PluginPropagationReconciler{}, false, ""),
		Entry("team", &greenhousev1alpha1.Team{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-team",
				Namespace:   test.TestNamespace,
				Labels:      map[string]string{"test-label": "test-value"},
				Annotations: map[string]string{"test-annotation": "test-value"},
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Team",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			Spec: greenhousev1alpha1.TeamSpec{
				MappedIDPGroup: "TEST_IDP_GROUP",
				Description:    "test description",
			},
		}, &team.TeamPropagationReconciler{}, false, ""),
		Entry("team membership", &greenhousev1alpha1.TeamMembership{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-team-membership",
				Namespace:   test.TestNamespace,
				Labels:      map[string]string{"test-label": "test-value"},
				Annotations: map[string]string{"test-annotation": "test-value"},
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "TeamMembership",
				APIVersion: greenhousev1alpha1.GroupVersion.String(),
			},
			Spec: greenhousev1alpha1.TeamMembershipSpec{
				Members: []greenhousev1alpha1.User{
					{
						ID:        "I12345",
						FirstName: "John",
						LastName:  "Doe",
						Email:     "john.doe@example.com",
					},
				},
			},
		}, &teammembership.TeamMembershipPropagationReconciler{}, false, ""),
		Entry("invalid object", &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-pod",
				Namespace:   test.TestNamespace,
				Labels:      map[string]string{"test-label": "test-value"},
				Annotations: map[string]string{"test-annotation": "test-value"},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "test-container",
						Image: "test-image",
					},
				},
			},
		}, &teammembership.TeamMembershipPropagationReconciler{}, true, "is not a team-membership"),
	)

	It("should have created the crd on the remote clusters with owner reference", func() {
		CRDList := &apiextensionsv1.CustomResourceDefinitionList{}
		otherCRDList := &apiextensionsv1.CustomResourceDefinitionList{}
		Eventually(func(g Gomega) bool {
			err := remoteClient.List(test.Ctx, CRDList)
			Expect(err).ToNot(HaveOccurred(), "there should be no error listing CRDs on the remote cluster")
			err = otherRemoteClient.List(test.Ctx, otherCRDList)
			Expect(err).ToNot(HaveOccurred(), "there should be no error listing CRDs on the other remote cluster")
			// repeat until assertion is true
			g.Expect(CRDList.Items).ToNot(BeEmpty(), "CRD list should not be empty")
			g.Expect(otherCRDList.Items).ToNot(BeEmpty(), "other CRD list should not be empty")
			g.Expect(CRDList.Items).To(ContainElement(test.ClientObjectMatcherByName(dummyCRDName)), "CRD should be present on the remote cluster")
			g.Expect(otherCRDList.Items).To(ContainElement(test.ClientObjectMatcherByName(dummyCRDName)), "CRD should be present on the other remote cluster")
			g.Expect(CRDList.Items[0].ObjectMeta.OwnerReferences[0].Name).To(Equal(test.TestNamespace))
			g.Expect(otherCRDList.Items[0].ObjectMeta.OwnerReferences[0].Name).To(Equal(test.TestNamespace))
			return true
		}).Should(BeTrue(), "CRD should be present")
	})

	It("should have created the resource on the remote cluster", func() {
		remoteObject := &fixtures.Dummy{}
		otherRemoteObject := &fixtures.Dummy{}
		Eventually(func(g Gomega) bool {
			err := remoteClient.Get(test.Ctx, types.NamespacedName{Namespace: dummy.GetNamespace(), Name: dummy.GetName()}, remoteObject)
			g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the resource on the remote cluster")
			err = otherRemoteClient.Get(test.Ctx, types.NamespacedName{Namespace: dummy.GetNamespace(), Name: dummy.GetName()}, otherRemoteObject)
			g.Expect(err).ToNot(HaveOccurred(), "there should be no error getting the resource on the other remote cluster")
			return true
		}).Should(BeTrue(), "there should be no error getting the remote cluster")
		expectObjectsToMatch(dummy, remoteObject)
		expectObjectsToMatch(dummy, otherRemoteObject)
	})

	It("should reconcile CRD and object after CRD update", func() {
		By("updating the crd in the local cluster")
		currentCRD := &apiextensionsv1.CustomResourceDefinition{}
		err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Namespace: "", Name: dummyCRDName}, currentCRD)
		Expect(err).To(Not(HaveOccurred()), "there should be no error getting the current CRD")
		crdUpdate, err := readCRDFromFile(filepath.Join(".", "fixtures", "dummy_crd_update.yaml"))
		Expect(err).To(Not(HaveOccurred()), "there should be no error reading the updated crd file")
		crdUpdate.ResourceVersion = currentCRD.ResourceVersion
		err = test.K8sClient.Update(test.Ctx, crdUpdate)
		Expect(err).To(Not(HaveOccurred()), "there should be no error updating the crd")

		By("checking the crd in the remote clusters")
		updatedCRD := &apiextensionsv1.CustomResourceDefinition{}
		otherUpdatedCRD := &apiextensionsv1.CustomResourceDefinition{}
		Eventually(func(g Gomega) bool {
			err = remoteClient.Get(test.Ctx, types.NamespacedName{Namespace: "", Name: crdUpdate.GetName()}, updatedCRD)
			Expect(err).ToNot(HaveOccurred(), "there should be no error listing CRDs on the remote cluster")
			g.Expect(updatedCRD.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties).ToNot(BeNil())
			g.Expect(updatedCRD.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties).To(HaveKey("secondProperty"), "CRD should be updated")
			err = otherRemoteClient.Get(test.Ctx, types.NamespacedName{Namespace: "", Name: crdUpdate.GetName()}, otherUpdatedCRD)
			Expect(err).ToNot(HaveOccurred(), "there should be no error listing CRDs on the other remote cluster")
			g.Expect(otherUpdatedCRD.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties).ToNot(BeNil())
			g.Expect(otherUpdatedCRD.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties).To(HaveKey("secondProperty"), "CRD should be updated")
			return true
		}).Should(BeTrue(), "CRD should be updated")

		By("checking the resources in the remote clusters")
		remoteObject := &fixtures.Dummy{}
		otherRemoteObject := &fixtures.Dummy{}
		Eventually(func(g Gomega) bool {
			err = remoteClient.Get(test.Ctx, types.NamespacedName{Name: dummy.GetName(), Namespace: dummy.GetNamespace()}, remoteObject)
			Expect(err).ToNot(HaveOccurred(), "there should be no error getting the unstructured remote object")
			Expect(remoteObject).NotTo(BeNil())
			Expect(remoteObject.Spec.SecondProperty).To(Equal("default"))
			err = otherRemoteClient.Get(test.Ctx, types.NamespacedName{Name: dummy.GetName(), Namespace: dummy.GetNamespace()}, otherRemoteObject)
			Expect(err).ToNot(HaveOccurred(), "there should be no error getting the unstructured other remote object")
			Expect(otherRemoteObject).NotTo(BeNil())
			Expect(otherRemoteObject.Spec.SecondProperty).To(Equal("default"))
			return true
		}).Should(BeTrue(), "the defaulted value should be on the remote resource")
	})

	It("should reconcile the remote resource after local resource update", func() {
		By("getting the local resource")
		localObject := dummy
		updateLabel := map[string]string{"test": "test"}
		err := test.K8sClient.Get(test.Ctx, types.NamespacedName{Namespace: dummy.GetNamespace(), Name: dummy.GetName()}, dummy)
		Expect(err).ToNot(HaveOccurred(), "there should be no error getting the local resource")
		Expect(localObject).NotTo(BeNil())

		By("updating the local resource")
		localObject.SetLabels(updateLabel)
		err = test.K8sClient.Update(test.Ctx, localObject)
		Expect(err).ToNot(HaveOccurred(), "there should be no error updating the local resource")

		remoteObject := &fixtures.Dummy{}
		otherRemoteObject := &fixtures.Dummy{}
		By("checking the remote resources")
		Eventually(func(g Gomega) bool {
			err = remoteClient.Get(test.Ctx, types.NamespacedName{Name: localObject.GetName(), Namespace: localObject.GetNamespace()}, remoteObject)
			Expect(err).ToNot(HaveOccurred(), "there should be no error getting the remote resource")
			g.Expect(remoteObject.GetLabels()).To(Equal(updateLabel), "the remote spec should have been updated")
			err = otherRemoteClient.Get(test.Ctx, types.NamespacedName{Name: localObject.GetName(), Namespace: localObject.GetNamespace()}, otherRemoteObject)
			Expect(err).ToNot(HaveOccurred(), "there should be no error getting the other remote resource")
			g.Expect(otherRemoteObject.GetLabels()).To(Equal(updateLabel), "the other remote spec should have been updated")
			return true
		}).Should(BeTrue(), "the remote spec should have been updated")

	})

	It("should delete the remote resources after deletion", func() {
		By("deleting the local resource")
		err := test.K8sClient.Delete(test.Ctx, dummy)
		Expect(err).ToNot(HaveOccurred(), "there should be no error deleting the resource")

		By("checking the remote resources")
		remoteObject := &fixtures.Dummy{}
		otherRemoteObject := &fixtures.Dummy{}
		Eventually(func(g Gomega) bool {
			err = remoteClient.Get(test.Ctx, types.NamespacedName{Name: dummy.GetName(), Namespace: dummy.GetNamespace()}, remoteObject)
			g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "the remote resource should have been deleted")
			err = otherRemoteClient.Get(test.Ctx, types.NamespacedName{Name: dummy.GetName(), Namespace: dummy.GetNamespace()}, otherRemoteObject)
			g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "the other remote resource should have been deleted")
			return true
		}).Should(BeTrue(), "the remote resource should have been deleted")

		By("checking the local resources")
		Eventually(func() bool {
			return apierrors.IsNotFound(test.K8sClient.Get(test.Ctx, types.NamespacedName{Name: dummy.GetName(), Namespace: dummy.GetNamespace()}, dummy))
		}).Should(BeTrue(), "the local resource should have been deleted")

	})

	// TODO test reconciling resources across multiple namespaces after CRD update

})

func readCRDFromFile(crdFile string) (*apiextensionsv1.CustomResourceDefinition, error) {
	crdFileData, err := os.ReadFile(crdFile)
	if err != nil {
		return nil, err
	}

	crd := &apiextensionsv1.CustomResourceDefinition{}
	scheme := runtime.NewScheme()
	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	decode := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode
	_, _, err = decode(crdFileData, nil, crd)
	if err != nil {
		return nil, err
	}
	return crd, nil
}

func expectObjectsToMatch(source, target client.Object) {
	Expect(source.GetName()).To(Equal(target.GetName()), "stripped object should have the same name")
	Expect(source.GetNamespace()).To(Equal(target.GetNamespace()), "stripped object should have the same namespace")
	Expect(source.GetLabels()).To(Equal(target.GetLabels()), "stripped object should have the same labels")
	Expect(source.GetAnnotations()).To(Equal(target.GetAnnotations()), "stripped object should have the same annotations")

	switch v := target.(type) {
	case *greenhousev1alpha1.Cluster:
		Expect(source.(*greenhousev1alpha1.Cluster).Spec).To(Equal(v.Spec), "stripped cluster should have the same spec")
		Expect(source).To(BeAssignableToTypeOf(target.(*greenhousev1alpha1.Cluster)), "stripped cluster should have underlying cluster type")
	case *greenhousev1alpha1.PluginDefinition:
		Expect(source.(*greenhousev1alpha1.PluginDefinition).Spec).To(Equal(v.Spec), "stripped pluginDefinitionDefinition should have the same spec")
		Expect(source).To(BeAssignableToTypeOf(target.(*greenhousev1alpha1.PluginDefinition)), "stripped pluginDefinition should have underlying pluginDefinition type")
	case *greenhousev1alpha1.Plugin:
		Expect(source.(*greenhousev1alpha1.Plugin).Spec).To(Equal(v.Spec), "stripped plugin should have the same spec")
		Expect(source).To(BeAssignableToTypeOf(target.(*greenhousev1alpha1.Plugin)), "stripped plugin should have underlying plugin type")
	case *greenhousev1alpha1.Team:
		Expect(source.(*greenhousev1alpha1.Team).Spec).To(Equal(v.Spec), "stripped team should have the same spec")
		Expect(source).To(BeAssignableToTypeOf(target.(*greenhousev1alpha1.Team)), "stripped team should have underlying team type")
	case *greenhousev1alpha1.TeamMembership:
		Expect(source.(*greenhousev1alpha1.TeamMembership).Spec).To(Equal(v.Spec), "stripped team membership should have the same spec")
		Expect(source).To(BeAssignableToTypeOf(target.(*greenhousev1alpha1.TeamMembership)), "stripped team membership should have underlying team membership type")
	}
}

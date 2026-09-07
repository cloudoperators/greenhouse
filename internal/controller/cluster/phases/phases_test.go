// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

func TestPhases(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cluster Phases Suite")
}

var _ = Describe("ensureNodesReady", func() {
	var (
		p       *Phase
		cluster *greenhousev1alpha1.Cluster
	)

	BeforeEach(func() {
		p = &Phase{}
		cluster = &greenhousev1alpha1.Cluster{}
	})

	Context("when cluster mode is Workerless", func() {
		BeforeEach(func() {
			cluster.Spec.Mode = greenhousev1alpha1.ClusterModeWorkerless
			// pre-set a stale AllNodesReady condition to verify it gets removed
			cluster.SetCondition(greenhousemetav1alpha1.TrueCondition(greenhousev1alpha1.AllNodesReady, "", ""))
			cluster.Status.Nodes = &greenhousev1alpha1.Nodes{Total: 3}
		})

		It("should remove the AllNodesReady condition", func() {
			_, err := p.ensureNodesReady(cluster)(context.Background())
			Expect(err).ToNot(HaveOccurred())
			Expect(cluster.Status.GetConditionByType(greenhousev1alpha1.AllNodesReady)).To(BeNil())
		})

		It("should nil out Status.Nodes", func() {
			_, err := p.ensureNodesReady(cluster)(context.Background())
			Expect(err).ToNot(HaveOccurred())
			Expect(cluster.Status.Nodes).To(BeNil())
		})
	})
})

var _ = Describe("ensureWorkloadSchedulable", func() {
	var (
		p       *Phase
		cluster *greenhousev1alpha1.Cluster
	)

	BeforeEach(func() {
		p = &Phase{}
		cluster = &greenhousev1alpha1.Cluster{}
	})

	Context("when cluster mode is Workerless", func() {
		BeforeEach(func() {
			cluster.Spec.Mode = greenhousev1alpha1.ClusterModeWorkerless
		})

		It("should set PayloadSchedulable=False with reason WorkerlessCluster", func() {
			_, err := p.ensureWorkloadSchedulable(cluster)(context.Background())
			Expect(err).ToNot(HaveOccurred())

			cond := cluster.Status.GetConditionByType(greenhousev1alpha1.PayloadSchedulable)
			Expect(cond).ToNot(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			Expect(cond.Reason).To(Equal(greenhousev1alpha1.WorkerlessClusterReason))
		})

		It("should not consult KubeConfigValid or AllNodesReady", func() {
			// both conditions absent — normal Default logic would set True; workerless must not reach that
			_, err := p.ensureWorkloadSchedulable(cluster)(context.Background())
			Expect(err).ToNot(HaveOccurred())

			cond := cluster.Status.GetConditionByType(greenhousev1alpha1.PayloadSchedulable)
			Expect(cond.Status).To(Equal(metav1.ConditionFalse))
		})
	})

	Context("when cluster mode is Default", func() {
		BeforeEach(func() {
			cluster.Spec.Mode = greenhousev1alpha1.ClusterModeDefault
		})

		It("should set PayloadSchedulable=True when all conditions pass", func() {
			_, err := p.ensureWorkloadSchedulable(cluster)(context.Background())
			Expect(err).ToNot(HaveOccurred())

			cond := cluster.Status.GetConditionByType(greenhousev1alpha1.PayloadSchedulable)
			Expect(cond).ToNot(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		})

		It("should set PayloadSchedulable=False when KubeConfigValid is False", func() {
			cluster.SetCondition(greenhousemetav1alpha1.FalseCondition(greenhousev1alpha1.KubeConfigValid, "", "cert expired"))

			_, err := p.ensureWorkloadSchedulable(cluster)(context.Background())
			Expect(err).ToNot(HaveOccurred())

			cond := cluster.Status.GetConditionByType(greenhousev1alpha1.PayloadSchedulable)
			Expect(cond).ToNot(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionFalse))
		})
	})
})

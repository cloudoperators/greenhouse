// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package admission

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	extensionsgreenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/extensions.greenhouse/v1alpha1"
)

var _ = Describe("Validate Update Rolebinding", func() {
	Context("ensures that changes to the immutable ClusterSelector are detected", func() {
		defaultSelector := metav1.LabelSelector{
			MatchLabels: map[string]string{
				"test.greenhouse.sap/cluster": "test-cluster",
			},
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "test.greenhouse.sap/zone",
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"test-zone"}},
			},
		}
		emptySelector := metav1.LabelSelector{}
		editedSelectorLabels := metav1.LabelSelector{
			MatchLabels: map[string]string{
				"test.greenhouse.sap/cluster": "edited-cluster",
			},
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "test.greenhouse.sap/zone",
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"test-zone"}},
			},
		}
		editedSelectorExpression := metav1.LabelSelector{
			MatchLabels: map[string]string{
				"test.greenhouse.sap/cluster": "test-cluster",
			},
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "test.greenhouse.sap/zone",
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"edited-zone"}},
			},
		}
		deletedSelectorLabels := metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "test.greenhouse.sap/zone",
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"test-zone"}},
			},
		}
		deletedSelectorExpressions := metav1.LabelSelector{
			MatchLabels: map[string]string{
				"test.greenhouse.sap/cluster": "test-cluster",
			},
		}

		DescribeTable("Validate that adding, removing, or editing ClusterSelector is detected", func(oldSelector, curSelector metav1.LabelSelector, expChange bool) {
			oldRB := &extensionsgreenhousev1alpha1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "greenhouse",
					Name:      "testBinding",
				},
				Spec: extensionsgreenhousev1alpha1.RoleBindingSpec{
					RoleRef:         "testRole",
					Namespaces:      []string{"testNamespace"},
					ClusterSelector: oldSelector,
				},
			}

			curRB := &extensionsgreenhousev1alpha1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "greenhouse",
					Name:      "testBinding",
				},
				Spec: extensionsgreenhousev1alpha1.RoleBindingSpec{
					RoleRef:         "testRole",
					Namespaces:      []string{"testNamespace"},
					ClusterSelector: curSelector,
				},
			}

			switch hasChanged := hasClusterSelectorChanged(oldRB, curRB); hasChanged {
			case true:
				Expect(expChange).To(BeTrue(), "expected ClusterSelector changes, but none found")
			default:
				Expect(expChange).To(BeFalse(), "unexpected ClusterSelector change detected")
			}
		},
			Entry("No Changes, all good", defaultSelector, defaultSelector, false),
			Entry("New selector added", emptySelector, defaultSelector, true),
			Entry("ClusterSelector removed", defaultSelector, emptySelector, true),
			Entry("Label Selector edited", defaultSelector, editedSelectorLabels, true),
			Entry("Expression Selector edited", defaultSelector, editedSelectorExpression, true),
			Entry("Label Selector deleted", defaultSelector, deletedSelectorLabels, true),
			Entry("Expression Selector deleted", defaultSelector, deletedSelectorExpressions, true),
		)
	})

	Context("ensures that changes to the immutable Namespaces are detected", func() {
		defaultNamespaces := []string{"testNamespace", "demoNamespace"}
		emptyNamespaces := []string{}
		editedNamespaces := []string{"editedNamespace", "demoNamespace"}
		deletedNamespaces := []string{"demoNamespace"}

		DescribeTable("Validate that adding, removing, or editing Namespaces is detected", func(oldNamespaces, curNamespaces []string, expChange bool) {
			oldRB := &extensionsgreenhousev1alpha1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "greenhouse",
					Name:      "testBinding",
				},
				Spec: extensionsgreenhousev1alpha1.RoleBindingSpec{
					RoleRef:    "testRole",
					Namespaces: oldNamespaces,
				},
			}

			curRB := &extensionsgreenhousev1alpha1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "greenhouse",
					Name:      "testBinding",
				},
				Spec: extensionsgreenhousev1alpha1.RoleBindingSpec{
					RoleRef:    "testRole",
					Namespaces: curNamespaces,
				},
			}

			switch hasChanged := hasNamespacesChanged(oldRB, curRB); hasChanged {
			case true:
				Expect(expChange).To(BeTrue(), "expected Namespaces changes, but none found")
			default:
				Expect(expChange).To(BeFalse(), "unexpected Namespaces change detected")
			}
		},
			Entry("No Changes, all good", defaultNamespaces, defaultNamespaces, false),
			Entry("Namespaces added", emptyNamespaces, defaultNamespaces, true),
			Entry("Namespaces removed", defaultNamespaces, emptyNamespaces, true),
			Entry("Namespaces edited", defaultNamespaces, editedNamespaces, true),
			Entry("Namespaces deleted", defaultNamespaces, deletedNamespaces, true),
		)
	})
})

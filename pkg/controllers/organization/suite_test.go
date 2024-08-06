// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudoperators/greenhouse/pkg/admission"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

func TestOrganization(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OrganizationControllerSuite")
}

var _ = BeforeSuite(func() {
	test.RegisterController("namespaceController", (&NamespaceReconciler{}).SetupWithManager)
	test.RegisterController("organizationController", (&RBACReconciler{}).SetupWithManager)
	test.RegisterWebhook("orgWebhook", admission.SetupOrganizationWebhookWithManager)
	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	test.TestAfterSuite()
})

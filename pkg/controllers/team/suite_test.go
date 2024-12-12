// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package team

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	greenhousecluster "github.com/cloudoperators/greenhouse/pkg/controllers/cluster"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

//+kubebuilder:rbac:groups=greenhouse.sap,resources=team,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=teammembership,verbs=get,list

func TestTeamController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TeamControllerSuite")
}

var _ = BeforeSuite(func() {
	test.RegisterController("teamController", (&TeamReconciler{}).SetupWithManager)
	test.RegisterController("cluster", (&greenhousecluster.RemoteClusterReconciler{}).SetupWithManager)

	test.TestBeforeSuite()
})

var _ = AfterSuite(func() {
	test.TestAfterSuite()
})

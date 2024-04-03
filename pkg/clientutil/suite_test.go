// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudoperators/greenhouse/pkg/admission"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

func TestClientUtil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client util Suite")
}

var _ = BeforeSuite(func() {
	test.RegisterWebhook("secretsWebhook", admission.SetupSecretWebhookWithManager)
	test.TestBeforeSuite()
	// return the test.Cfg, as the in-cluster config is not available
	ctrl.GetConfig = func() (*rest.Config, error) {
		return test.Cfg, nil
	}
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	test.TestAfterSuite()
})

func returnTestKubeConfigSecret(secretType corev1.SecretType, dataKey string, kubeConfig []byte) corev1.Secret {
	return corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.GroupName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: corev1.NamespaceDefault,
		},
		Data: map[string][]byte{
			dataKey: kubeConfig,
		},
		Type: secretType,
	}
}

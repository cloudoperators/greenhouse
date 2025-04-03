// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
)

const (
	OIDCSecretResource  = "oidc-secret"
	OIDCClientIDKey     = "clientID"
	OIDCClientID        = "the-client-id"
	OIDCClientSecretKey = "clientSecret"
	OIDCClientSecret    = "the-client-secret"
	OIDCIssuer          = "https://the-issuer"
)

type TestSetup struct {
	client.Client
	namespace string
}

func (t *TestSetup) Namespace() string {
	return t.namespace
}

// RandomizeName returns the name with a random alphanumeric suffix
func (t *TestSetup) RandomizeName(name string) string {
	return name + "-" + rand.String(8)
}

// NewTestSetup creates a new TestSetup object and a new namespace on the cluster for the test
func NewTestSetup(ctx context.Context, c client.Client, name string) *TestSetup {
	suffix := rand.String(8)

	t := &TestSetup{
		Client:    c,
		namespace: name + "-" + suffix,
	}

	// Create test namespace
	Expect(t.Create(Ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: t.namespace}})).To(Succeed(), "there should be no error creating the test case namespace")
	return t
}

// OnboardCluster creates a new Cluster and Kubernetes secret for a remote cluster and creates the namespace used for TestSetup on the remote cluster
func (t *TestSetup) OnboardCluster(ctx context.Context, name string, kubeCfg []byte, opts ...func(*greenhousev1alpha1.Cluster)) *greenhousev1alpha1.Cluster {
	GinkgoHelper()
	cluster := t.CreateCluster(ctx, name, opts...)

	var testClusterK8sSecret = &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.GroupName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: t.Namespace(),
		},
		Type: greenhouseapis.SecretTypeKubeConfig,
		Data: map[string][]byte{
			greenhouseapis.GreenHouseKubeConfigKey: kubeCfg,
		},
	}
	Expect(t.Create(ctx, testClusterK8sSecret)).Should(Succeed(), "there should be no error creating the kubeconfig secret during onboarding")

	restClientGetter, err := clientutil.NewRestClientGetterFromSecret(testClusterK8sSecret, t.Namespace())
	Expect(err).NotTo(HaveOccurred(), "there should be no error creating the rest client getter from the kubeconfig secret during onboarding")

	k8sClientForRemoteCluster, err := clientutil.NewK8sClientFromRestClientGetter(restClientGetter)
	Expect(err).NotTo(HaveOccurred(), "there should be no error creating the k8s client from the rest client getter during onboarding")

	var namespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: t.Namespace(),
		}}
	Expect(k8sClientForRemoteCluster.Create(ctx, namespace)).To(Succeed(), "there should be no error creating the namespace during onboarding")

	return cluster
}

// CreateCluster creates a new Cluster resource without creating a Secret
func (t *TestSetup) CreateCluster(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.Cluster)) *greenhousev1alpha1.Cluster {
	GinkgoHelper()
	cluster := NewCluster(ctx, name, t.Namespace(), opts...)
	Expect(t.Create(ctx, cluster)).To(Succeed(), "there should be no error creating the cluster during onboarding")
	return cluster
}

func (t *TestSetup) CreateOrganizationWithOIDCConfig(ctx context.Context, orgName string) (*greenhousev1alpha1.Organization, *corev1.Secret) {
	GinkgoHelper()
	secret := t.CreateSecret(ctx, OIDCSecretResource,
		WithSecretNamespace(orgName),
		WithSecretData(map[string][]byte{
			OIDCClientIDKey:     []byte(OIDCClientID),
			OIDCClientSecretKey: []byte(OIDCClientSecret),
		}))

	org := t.CreateOrganization(ctx, orgName, WithOIDCConfig(OIDCIssuer, secret.Name, OIDCClientIDKey, OIDCClientSecretKey))
	return org, secret
}

// CreateOrganization creates a Organization within the TestSetup and returns the created Organization resource.
func (t *TestSetup) CreateOrganization(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.Organization)) *greenhousev1alpha1.Organization {
	GinkgoHelper()
	org := NewOrganization(ctx, name, opts...)
	Expect(t.Create(ctx, org)).Should(Succeed(), "there should be no error creating the Organization")
	return org
}

// CreatePluginDefinition creates and returns a PluginDefinition object. Opts can be used to set the desired state of the PluginDefinition.
func (t *TestSetup) CreatePluginDefinition(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.PluginDefinition)) *greenhousev1alpha1.PluginDefinition {
	GinkgoHelper()
	pd := NewPluginDefinition(ctx, t.RandomizeName(name), t.Namespace(), opts...)
	Expect(t.Create(ctx, pd)).Should(Succeed(), "there should be no error creating the PluginDefinition")
	return pd
}

// CreatePlugin creates and returns a Plugin object. Opts can be used to set the desired state of the Plugin.
func (t *TestSetup) CreatePlugin(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.Plugin)) *greenhousev1alpha1.Plugin {
	GinkgoHelper()
	plugin := NewPlugin(ctx, name, t.Namespace(), opts...)
	Expect(t.Create(ctx, plugin)).Should(Succeed(), "there should be no error creating the Plugin")
	return plugin
}

// CreateTeamRole returns a TeamRole object. Opts can be used to set the desired state of the TeamRole.
func (t *TestSetup) CreateTeamRole(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.TeamRole)) *greenhousev1alpha1.TeamRole {
	GinkgoHelper()
	tr := NewTeamRole(ctx, t.RandomizeName(name), t.namespace, opts...)
	Expect(t.Create(ctx, tr)).Should(Succeed(), "there should be no error creating the TeamRole")
	return tr
}

// CreateTeamRoleBinding returns a TeamRoleBinding object. Opts can be used to set the desired state of the TeamRoleBinding.
func (t *TestSetup) CreateTeamRoleBinding(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.TeamRoleBinding)) *greenhousev1alpha1.TeamRoleBinding {
	GinkgoHelper()
	trb := NewTeamRoleBinding(ctx, t.RandomizeName(name), t.Namespace(), opts...)
	Expect(t.Create(ctx, trb)).Should(Succeed(), "there should be no error creating the TeamRoleBinding")
	return trb
}

// CreateTeam returns a Team object. Opts can be used to set the desired state of the Team.st
func (t *TestSetup) CreateTeam(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.Team)) *greenhousev1alpha1.Team {
	GinkgoHelper()
	team := NewTeam(ctx, name, t.Namespace(), opts...)
	Expect(t.Create(ctx, team)).Should(Succeed(), "there should be no error creating the Team")
	return team
}

// CreateSecret returns a Secret object. Opts can be used to set the desired state of the Secret.
func (t *TestSetup) CreateSecret(ctx context.Context, name string, opts ...func(*corev1.Secret)) *corev1.Secret {
	GinkgoHelper()
	secret := NewSecret(ctx, name, t.Namespace(), opts...)
	Expect(t.Create(ctx, secret)).Should(Succeed(), "there should be no error creating the Secret")
	return secret
}

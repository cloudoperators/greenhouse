// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
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

func (t *TestSetup) CreateOrganizationWithOIDCConfig(ctx context.Context, orgName, supportGroupTeamName string) (*greenhousev1alpha1.Organization, *corev1.Secret) {
	GinkgoHelper()
	secret := t.CreateOrgOIDCSecret(ctx, orgName, supportGroupTeamName)
	org := t.CreateOrganization(ctx, orgName, WithMappedAdminIDPGroup(orgName+" Admin E2e"), WithOIDCConfig(OIDCIssuer, secret.Name, OIDCClientIDKey, OIDCClientSecretKey))
	return org, secret
}

func (t *TestSetup) CreateOrgOIDCSecret(ctx context.Context, orgName, supportGroupTeamName string) *corev1.Secret {
	GinkgoHelper()
	secret := t.CreateSecret(ctx, OIDCSecretResource,
		WithSecretNamespace(orgName),
		WithSecretData(map[string][]byte{
			OIDCClientIDKey:     []byte(OIDCClientID),
			OIDCClientSecretKey: []byte(OIDCClientSecret),
		}),
		WithSecretLabel(greenhouseapis.LabelKeyOwnedBy, supportGroupTeamName))
	return secret
}

// CreateOrganization creates an Organization within the TestSetup and returns the created Organization resource.
func (t *TestSetup) CreateOrganization(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.Organization)) *greenhousev1alpha1.Organization {
	GinkgoHelper()
	org := NewOrganization(ctx, name, opts...)
	Expect(t.Create(ctx, org)).Should(Succeed(), "there should be no error creating the Organization")
	return org
}

func (t *TestSetup) CreateDefaultOrgWithOIDCSecret(ctx context.Context, supportGroupTeamName string) *greenhousev1alpha1.Organization {
	GinkgoHelper()
	org := &greenhousev1alpha1.Organization{}
	err := t.Get(ctx, client.ObjectKey{Name: "greenhouse"}, org)
	if err != nil {
		if apierrors.IsNotFound(err) {
			org = NewOrganization(ctx, "greenhouse", WithMappedAdminIDPGroup("Greenhouse Admin E2e"))
			Expect(t.Create(ctx, org)).Should(Succeed(), "there should be no error creating the default organization")
			EventuallyCreated(ctx, t.Client, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: org.Name}})
			secret := t.CreateOrgOIDCSecret(ctx, org.Name, supportGroupTeamName)
			org = t.UpdateOrganization(ctx, org.Name, WithOIDCConfig(OIDCIssuer, secret.Name, OIDCClientIDKey, OIDCClientSecretKey))
			return org
		}
	}
	Expect(err).NotTo(HaveOccurred(), "there should be no error getting the default organization")
	return org
}

func (t *TestSetup) UpdateOrganization(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.Organization)) *greenhousev1alpha1.Organization {
	GinkgoHelper()
	org := &greenhousev1alpha1.Organization{}
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := t.Get(ctx, client.ObjectKey{Name: name}, org)
		if err != nil {
			return err
		}
		for _, opt := range opts {
			opt(org)
		}
		return t.Update(ctx, org)
	})
	Expect(err).NotTo(HaveOccurred(), "there should be no error updating the Organization")
	return org
}

// CreateClusterPluginDefinition creates and returns a ClusterPluginDefinition object. Opts can be used to set the desired state of the ClusterPluginDefinition.
func (t *TestSetup) CreateClusterPluginDefinition(ctx context.Context, name string, opts ...func(definition *greenhousev1alpha1.ClusterPluginDefinition)) *greenhousev1alpha1.ClusterPluginDefinition {
	GinkgoHelper()
	pd := NewClusterPluginDefinition(ctx, t.RandomizeName(name), opts...)
	Expect(t.Create(ctx, pd)).Should(Succeed(), "there should be no error creating the ClusterPluginDefinition")
	return pd
}

// CreatePluginDefinition creates and returns a PluginDefinition object. Opts can be used to set the desired state of the PluginDefinition.
func (t *TestSetup) CreatePluginDefinition(ctx context.Context, name string, opts ...func(definition *greenhousev1alpha1.PluginDefinition)) *greenhousev1alpha1.PluginDefinition {
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

// CreatePluginPreset creates and returns a PluginPreset object. Opts can be used to set the desired state of the PluginPreset.
func (t *TestSetup) CreatePluginPreset(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.PluginPreset)) *greenhousev1alpha1.PluginPreset {
	GinkgoHelper()
	plugin := NewPluginPreset(name, t.Namespace(), opts...)
	Expect(t.Create(ctx, plugin)).Should(Succeed(), "there should be no error creating the PluginPreset")
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
func (t *TestSetup) CreateTeamRoleBinding(ctx context.Context, name string, opts ...func(*greenhousev1alpha2.TeamRoleBinding)) *greenhousev1alpha2.TeamRoleBinding {
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
	secret := NewSecret(name, t.Namespace(), opts...)
	Expect(t.Create(ctx, secret)).Should(Succeed(), "there should be no error creating the Secret")
	return secret
}

// UpdateSecret updates a Secret object. Opts can be used to set the desired state of the Secret.
func (t *TestSetup) UpdateSecret(ctx context.Context, name string, opts ...func(*corev1.Secret)) *corev1.Secret {
	GinkgoHelper()
	secret := &corev1.Secret{}
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := t.Get(ctx, client.ObjectKey{Name: name, Namespace: t.Namespace()}, secret)
		if err != nil {
			return err
		}
		for _, opt := range opts {
			opt(secret)
		}
		return t.Update(ctx, secret)
	})
	Expect(err).NotTo(HaveOccurred(), "there should be no error updating the Secret")
	return secret
}

// CreateConfigMap returns a ConfigMap object. Opts can be used to set the desired state of the ConfigMap.
func (t *TestSetup) CreateConfigMap(ctx context.Context, name string, opts ...func(*corev1.ConfigMap)) *corev1.ConfigMap {
	GinkgoHelper()
	cm := NewConfigMap(name, t.Namespace(), opts...)
	Expect(t.Create(ctx, cm)).Should(Succeed(), "there should be no error creating the ConfigMap")
	return cm
}

func (t *TestSetup) CreateCatalog(ctx context.Context, name string, opts ...func(catalog *greenhousev1alpha1.Catalog)) *greenhousev1alpha1.Catalog {
	GinkgoHelper()
	ns := t.Namespace()
	catalog := NewCatalog(name, ns, opts...)
	Expect(t.Create(ctx, catalog)).Should(Succeed(), "there should be no error creating the Catalog")
	return catalog
}

func (t *TestSetup) UpdateCatalog(ctx context.Context, name string, opts ...func(catalog *greenhousev1alpha1.Catalog)) *greenhousev1alpha1.Catalog {
	GinkgoHelper()
	catalog := &greenhousev1alpha1.Catalog{}
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ns := t.Namespace()
		err := t.Get(ctx, client.ObjectKey{Name: name, Namespace: ns}, catalog)
		if err != nil {
			return err
		}
		for _, opt := range opts {
			opt(catalog)
		}
		return t.Update(ctx, catalog)
	})
	Expect(err).NotTo(HaveOccurred(), "there should be no error updating the Organization")
	return catalog
}

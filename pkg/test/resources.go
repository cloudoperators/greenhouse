// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
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

// WithAccessMode sets the ClusterAccessMode on a Cluster
func WithAccessMode(mode greenhousev1alpha1.ClusterAccessMode) func(*greenhousev1alpha1.Cluster) {
	return func(c *greenhousev1alpha1.Cluster) {
		c.Spec.AccessMode = mode
	}
}

// WithLabel sets the label on a Cluster
func WithLabel(key, value string) func(*greenhousev1alpha1.Cluster) {
	return func(c *greenhousev1alpha1.Cluster) {
		if c.Labels == nil {
			c.Labels = make(map[string]string, 1)
		}
		c.Labels[key] = value
	}
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
	cluster := &greenhousev1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: t.Namespace(),
		},
		Spec: greenhousev1alpha1.ClusterSpec{
			AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
		},
	}

	for _, o := range opts {
		o(cluster)
	}

	Expect(t.Create(ctx, cluster)).To(Succeed(), "there should be no error creating the cluster during onboarding")
	return cluster
}

// CreateOrganization creates a Organization within the TestSetup and returns the created Organization resource.
func (t *TestSetup) CreateOrganization(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.Organization)) *greenhousev1alpha1.Organization {
	GinkgoHelper()
	org := &greenhousev1alpha1.Organization{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Organization",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	for _, o := range opts {
		o(org)
	}
	Expect(t.Create(ctx, org)).Should(Succeed(), "there should be no error creating the Organization")
	return org
}

// WithVersion sets the version of a PluginDefinition
func WithVersion(version string) func(*greenhousev1alpha1.PluginDefinition) {
	return func(pd *greenhousev1alpha1.PluginDefinition) {
		pd.Spec.Version = version
	}
}

// WithHelmChart sets the HelmChart of a PluginDefinition
func WithHelmChart(chart *greenhousev1alpha1.HelmChartReference) func(*greenhousev1alpha1.PluginDefinition) {
	return func(pd *greenhousev1alpha1.PluginDefinition) {
		pd.Spec.HelmChart = chart
	}
}

// AppendPluginOption sets the plugin option in plugin definition
func AppendPluginOption(option greenhousev1alpha1.PluginOption) func(*greenhousev1alpha1.PluginDefinition) {
	return func(pd *greenhousev1alpha1.PluginDefinition) {
		pd.Spec.Options = append(pd.Spec.Options, option)
	}
}

// CreatePluginDefinition creates and returns a PluginDefinition object. Opts can be used to set the desired state of the PluginDefinition.
func (t *TestSetup) CreatePluginDefinition(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.PluginDefinition)) *greenhousev1alpha1.PluginDefinition {
	GinkgoHelper()
	pd := &greenhousev1alpha1.PluginDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PluginDefinition",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.RandomizeName(name),
			Namespace: t.Namespace(),
		},
		Spec: greenhousev1alpha1.PluginDefinitionSpec{
			Description: "TestPluginDefinition",
			Version:     "1.0.0",
			HelmChart: &greenhousev1alpha1.HelmChartReference{
				Name:       "./../../test/fixtures/myChart",
				Repository: "dummy",
				Version:    "1.0.0",
			},
		},
	}
	for _, o := range opts {
		o(pd)
	}

	Expect(t.Create(ctx, pd)).Should(Succeed(), "there should be no error creating the PluginDefinition")
	return pd
}

// WithPluginDefinition sets the PluginDefinition of a Plugin
func WithPluginDefinition(pluginDefinition string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		p.Spec.PluginDefinition = pluginDefinition
	}
}

// WithReleaseNamespace sets the ReleaseNamespace of a Plugin
func WithReleaseNamespace(releaseNamespace string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		p.Spec.ReleaseNamespace = releaseNamespace
	}
}

// WithCluster sets the Cluster for a Plugin
func WithCluster(cluster string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		p.Spec.ClusterName = cluster
	}
}

// WithPluginOptionValue sets the value of a PluginOptionValue
func WithPluginOptionValue(name string, value *apiextensionsv1.JSON, valueFrom *greenhousev1alpha1.ValueFromSource) func(*greenhousev1alpha1.Plugin) {
	GinkgoHelper()
	return func(p *greenhousev1alpha1.Plugin) {
		if value != nil && valueFrom != nil {
			Fail("value and valueFrom are mutually exclusive")
		}
		for i, v := range p.Spec.OptionValues {
			if v.Name == name {
				v.Value = value
				v.ValueFrom = valueFrom
				p.Spec.OptionValues[i] = v
				return
			}
			p.Spec.OptionValues = append(p.Spec.OptionValues, greenhousev1alpha1.PluginOptionValue{
				Name:      name,
				Value:     value,
				ValueFrom: valueFrom,
			})
		}
	}
}

// SetOptionValueForPlugin sets the value of a PluginOptionValue in plugin
func SetOptionValueForPlugin(plugin *greenhousev1alpha1.Plugin, key, value string) {
	for i, keyValue := range plugin.Spec.OptionValues {
		if keyValue.Name == key {
			plugin.Spec.OptionValues[i].Value.Raw = []byte(value)
			return
		}
	}

	plugin.Spec.OptionValues = append(plugin.Spec.OptionValues, greenhousev1alpha1.PluginOptionValue{
		Name:  key,
		Value: &apiextensionsv1.JSON{Raw: []byte(value)},
	})
}

// NewPlugin creates new plugin
func (t *TestSetup) NewPlugin(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.Plugin)) *greenhousev1alpha1.Plugin {
	GinkgoHelper()
	plugin := &greenhousev1alpha1.Plugin{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Plugin",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.RandomizeName(name),
			Namespace: t.Namespace(),
		},
	}
	for _, o := range opts {
		o(plugin)
	}
	return plugin
}

// CreatePlugin creates and returns a Plugin object. Opts can be used to set the desired state of the Plugin.
func (t *TestSetup) CreatePlugin(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.Plugin)) *greenhousev1alpha1.Plugin {
	GinkgoHelper()
	plugin := t.NewPlugin(ctx, name, opts...)
	Expect(t.Create(ctx, plugin)).Should(Succeed(), "there should be no error creating the Plugin")
	return plugin
}

// WithRules overrides the default rules of a TeamRole
func WithRules(rules []rbacv1.PolicyRule) func(*greenhousev1alpha1.TeamRole) {
	return func(tr *greenhousev1alpha1.TeamRole) {
		tr.Spec.Rules = rules
	}
}

// WithAggregationRule sets the AggregationRule on a TeamRole
func WithAggregationRule(aggregationRule *rbacv1.AggregationRule) func(*greenhousev1alpha1.TeamRole) {
	return func(tr *greenhousev1alpha1.TeamRole) {
		tr.Spec.AggregationRule = aggregationRule
	}
}

// WithLabels sets the .spec.Labels on a TeamRole
func WithLabels(labels map[string]string) func(*greenhousev1alpha1.TeamRole) {
	return func(tr *greenhousev1alpha1.TeamRole) {
		tr.Spec.Labels = labels
	}
}

func (t *TestSetup) NewTeamRole(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.TeamRole)) *greenhousev1alpha1.TeamRole {
	tr := &greenhousev1alpha1.TeamRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TeamRole",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.RandomizeName(name),
			Namespace: t.Namespace(),
		},
		Spec: greenhousev1alpha1.TeamRoleSpec{
			Rules: []rbacv1.PolicyRule{
				{
					Verbs:     []string{"get"},
					APIGroups: []string{"*"},
					Resources: []string{"*"},
				},
			},
		},
	}
	for _, opt := range opts {
		opt(tr)
	}
	return tr
}

// CreateTeamRole returns a TeamRole object. Opts can be used to set the desired state of the TeamRole.
func (t *TestSetup) CreateTeamRole(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.TeamRole)) *greenhousev1alpha1.TeamRole {
	GinkgoHelper()
	tr := t.NewTeamRole(ctx, name, opts...)
	Expect(t.Create(ctx, tr)).Should(Succeed(), "there should be no error creating the TeamRole")
	return tr
}

func WithTeamRoleRef(roleRef string) func(*greenhousev1alpha1.TeamRoleBinding) {
	return func(trb *greenhousev1alpha1.TeamRoleBinding) {
		trb.Spec.TeamRoleRef = roleRef
	}
}

func WithTeamRef(teamRef string) func(*greenhousev1alpha1.TeamRoleBinding) {
	return func(trb *greenhousev1alpha1.TeamRoleBinding) {
		trb.Spec.TeamRef = teamRef
	}
}

func WithClusterName(clusterName string) func(*greenhousev1alpha1.TeamRoleBinding) {
	return func(trb *greenhousev1alpha1.TeamRoleBinding) {
		trb.Spec.ClusterName = clusterName
	}
}

func WithClusterSelector(selector metav1.LabelSelector) func(*greenhousev1alpha1.TeamRoleBinding) {
	return func(trb *greenhousev1alpha1.TeamRoleBinding) {
		trb.Spec.ClusterSelector = selector
	}
}

func WithNamespaces(namespaces ...string) func(*greenhousev1alpha1.TeamRoleBinding) {
	return func(trb *greenhousev1alpha1.TeamRoleBinding) {
		trb.Spec.Namespaces = namespaces
	}
}

// CreateTeamRoleBinding returns a TeamRoleBinding object. Opts can be used to set the desired state of the TeamRoleBinding.
func (t *TestSetup) CreateTeamRoleBinding(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.TeamRoleBinding)) *greenhousev1alpha1.TeamRoleBinding {
	GinkgoHelper()
	trb := &greenhousev1alpha1.TeamRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TeamRoleBinding",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.RandomizeName(name),
			Namespace: t.Namespace(),
		},
		Spec: greenhousev1alpha1.TeamRoleBindingSpec{},
	}
	for _, o := range opts {
		o(trb)
	}

	Expect(t.Create(ctx, trb)).Should(Succeed(), "there should be no error creating the TeamRoleBinding")
	return trb
}

func WithMappedIDPGroup(group string) func(*greenhousev1alpha1.Team) {
	return func(t *greenhousev1alpha1.Team) {
		t.Spec.MappedIDPGroup = group
	}
}

// CreateTeam returns a Team object. Opts can be used to set the desired state of the Team.st
func (t *TestSetup) CreateTeam(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.Team)) *greenhousev1alpha1.Team {
	GinkgoHelper()
	team := &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Team",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.RandomizeName(name),
			Namespace: t.Namespace(),
		},
	}
	for _, opt := range opts {
		opt(team)
	}
	Expect(t.Create(ctx, team)).Should(Succeed(), "there should be no error creating the Team")
	return team
}

// WithSecretType sets the type of the Secret
func WithSecretType(secretType corev1.SecretType) func(*corev1.Secret) {
	return func(s *corev1.Secret) {
		s.Type = secretType
	}
}

// WithSecretData sets the data of the Secret
func WithSecretData(data map[string][]byte) func(*corev1.Secret) {
	return func(s *corev1.Secret) {
		s.Data = data
	}
}

// CreateSecret returns a Secret object. Opts can be used to set the desired state of the Secret.
func (t *TestSetup) CreateSecret(ctx context.Context, name string, opts ...func(*corev1.Secret)) *corev1.Secret {
	GinkgoHelper()
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.GroupName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: t.Namespace(),
		},
	}
	for _, opt := range opts {
		opt(secret)
	}
	Expect(t.Create(ctx, secret)).Should(Succeed(), "there should be no error creating the Secret")
	return secret
}

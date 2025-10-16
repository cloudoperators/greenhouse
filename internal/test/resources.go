// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	"k8s.io/utils/ptr"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	greenhousev1alpha2 "github.com/cloudoperators/greenhouse/api/v1alpha2"
)

// WithAccessMode sets the ClusterAccessMode on a Cluster
func WithAccessMode(mode greenhousev1alpha1.ClusterAccessMode) func(*greenhousev1alpha1.Cluster) {
	return func(c *greenhousev1alpha1.Cluster) {
		c.Spec.AccessMode = mode
	}
}

// WithClusterLabel sets the label on a Cluster
func WithClusterLabel(key, value string) func(*greenhousev1alpha1.Cluster) {
	return func(c *greenhousev1alpha1.Cluster) {
		if c.Labels == nil {
			c.Labels = make(map[string]string, 1)
		}
		c.Labels[key] = value
	}
}

// WithClusterAnnotations sets metadata annotations on a Cluster
func WithClusterAnnotations(annotations map[string]string) func(*greenhousev1alpha1.Cluster) {
	return func(c *greenhousev1alpha1.Cluster) {
		c.SetAnnotations(annotations)
	}
}

// WithKubeConfig sets the kubeconfig of a Cluster
func WithMaxTokenValidity(maxTokenValidity int32) func(*greenhousev1alpha1.Cluster) {
	return func(c *greenhousev1alpha1.Cluster) {
		c.Spec.KubeConfig.MaxTokenValidity = maxTokenValidity
	}
}

// NewCluster returns a greenhousev1alpha1.Cluster object. Opts can be used to set the desired state of the Cluster.
func NewCluster(ctx context.Context, name, namespace string, opts ...func(*greenhousev1alpha1.Cluster)) *greenhousev1alpha1.Cluster {
	cluster := &greenhousev1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: greenhousev1alpha1.ClusterSpec{
			AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
		},
	}

	for _, o := range opts {
		o(cluster)
	}
	return cluster
}

// WithMappedAdminIDPGroup sets the MappedIDPGroup on an Organization
func WithMappedAdminIDPGroup(group string) func(*greenhousev1alpha1.Organization) {
	return func(org *greenhousev1alpha1.Organization) {
		org.Spec.MappedOrgAdminIDPGroup = group
	}
}

func WithOrgAnnotations(annotations map[string]string) func(*greenhousev1alpha1.Organization) {
	return func(org *greenhousev1alpha1.Organization) {
		org.SetAnnotations(annotations)
	}
}

// WithAdditionalRedirects - sets the additional redirect URIs on an Organization. (To be used with WithOIDCConfig)
func WithAdditionalRedirects(additionalRedirects ...string) func(organization *greenhousev1alpha1.Organization) {
	return func(org *greenhousev1alpha1.Organization) {
		if org.Spec.Authentication == nil {
			org.Spec.Authentication = &greenhousev1alpha1.Authentication{}
		}
		if org.Spec.Authentication.OIDCConfig == nil {
			org.Spec.Authentication.OIDCConfig = &greenhousev1alpha1.OIDCConfig{}
		}
		org.Spec.Authentication.OIDCConfig.OAuth2ClientRedirectURIs = additionalRedirects
	}
}

func WithConfigMapRef(configMapRef string) func(*greenhousev1alpha1.Organization) {
	return func(org *greenhousev1alpha1.Organization) {
		org.Spec.ConfigMapRef = configMapRef
	}
}

// WithOIDCConfig sets the OIDCConfig on an Organization
func WithOIDCConfig(issuer, secretName, clientIDKey, clientSecretKey string) func(*greenhousev1alpha1.Organization) {
	return func(org *greenhousev1alpha1.Organization) {
		if org.Spec.Authentication == nil {
			org.Spec.Authentication = &greenhousev1alpha1.Authentication{}
		}
		org.Spec.Authentication.OIDCConfig = &greenhousev1alpha1.OIDCConfig{
			Issuer: issuer,
			ClientIDReference: greenhousev1alpha1.SecretKeyReference{
				Name: secretName,
				Key:  clientIDKey,
			},
			ClientSecretReference: greenhousev1alpha1.SecretKeyReference{
				Name: secretName,
				Key:  clientSecretKey,
			},
			RedirectURI: issuer + "/callback",
		}
	}
}

// NewOrganization returns a greenhousev1alpha1.Organization object. Opts can be used to set the desired state of the Organization.
func NewOrganization(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.Organization)) *greenhousev1alpha1.Organization {
	org := &greenhousev1alpha1.Organization{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Organization",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: greenhousev1alpha1.OrganizationSpec{
			MappedOrgAdminIDPGroup: "default-admin-id-group",
		},
	}
	for _, o := range opts {
		o(org)
	}
	return org
}

// WithVersion sets the version of a ClusterPluginDefinition
func WithVersion(version string) func(*greenhousev1alpha1.ClusterPluginDefinition) {
	return func(pd *greenhousev1alpha1.ClusterPluginDefinition) {
		pd.Spec.Version = version
	}
}

// WithHelmChart sets the HelmChart of a ClusterPluginDefinition
func WithHelmChart(chart *greenhousev1alpha1.HelmChartReference) func(*greenhousev1alpha1.ClusterPluginDefinition) {
	return func(pd *greenhousev1alpha1.ClusterPluginDefinition) {
		pd.Spec.HelmChart = chart
	}
}

// WithoutHelmChart sets the HelmChart of a ClusterPluginDefinition to nil
func WithoutHelmChart() func(*greenhousev1alpha1.ClusterPluginDefinition) {
	return func(pd *greenhousev1alpha1.ClusterPluginDefinition) {
		pd.Spec.HelmChart = nil
	}
}

// WithDescription sets the description of a ClusterPluginDefinition
func WithUIApplication(ui *greenhousev1alpha1.UIApplicationReference) func(*greenhousev1alpha1.ClusterPluginDefinition) {
	return func(pd *greenhousev1alpha1.ClusterPluginDefinition) {
		pd.Spec.UIApplication = ui
	}
}

// AppendPluginOption sets the plugin option in ClusterPluginDefinition
func AppendPluginOption(option greenhousev1alpha1.PluginOption) func(*greenhousev1alpha1.ClusterPluginDefinition) {
	return func(pd *greenhousev1alpha1.ClusterPluginDefinition) {
		pd.Spec.Options = append(pd.Spec.Options, option)
	}
}

// NewClusterPluginDefinition returns a greenhousev1alpha1.ClusterPluginDefinition object. Opts can be used to set the desired state of the ClusterPluginDefinition.
func NewClusterPluginDefinition(ctx context.Context, name string, opts ...func(definition *greenhousev1alpha1.ClusterPluginDefinition)) *greenhousev1alpha1.ClusterPluginDefinition {
	pd := &greenhousev1alpha1.ClusterPluginDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
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
	return pd
}

// WithPluginDefinitionVersion sets the version of a ClusterPluginDefinition
func WithPluginDefinitionVersion(version string) func(*greenhousev1alpha1.PluginDefinition) {
	return func(pd *greenhousev1alpha1.PluginDefinition) {
		pd.Spec.Version = version
	}
}

// WithPluginDefinitionHelmChart sets the HelmChart of a ClusterPluginDefinition
func WithPluginDefinitionHelmChart(chart *greenhousev1alpha1.HelmChartReference) func(*greenhousev1alpha1.PluginDefinition) {
	return func(pd *greenhousev1alpha1.PluginDefinition) {
		pd.Spec.HelmChart = chart
	}
}

// WithoutPluginDefinitionHelmChart sets the HelmChart of a ClusterPluginDefinition to nil
func WithoutPluginDefinitionHelmChart() func(*greenhousev1alpha1.PluginDefinition) {
	return func(pd *greenhousev1alpha1.PluginDefinition) {
		pd.Spec.HelmChart = nil
	}
}

// WithPluginDefinitionUIApplication sets the description of a ClusterPluginDefinition
func WithPluginDefinitionUIApplication(ui *greenhousev1alpha1.UIApplicationReference) func(*greenhousev1alpha1.PluginDefinition) {
	return func(pd *greenhousev1alpha1.PluginDefinition) {
		pd.Spec.UIApplication = ui
	}
}

// AppendPluginDefinitionPluginOption sets the plugin option in ClusterPluginDefinition
func AppendPluginDefinitionPluginOption(option greenhousev1alpha1.PluginOption) func(*greenhousev1alpha1.PluginDefinition) {
	return func(pd *greenhousev1alpha1.PluginDefinition) {
		pd.Spec.Options = append(pd.Spec.Options, option)
	}
}

// NewPluginDefinition returns a namespaced greenhousev1alpha1.PluginDefinition object. Opts can be used to set the desired state of the PluginDefinition.
func NewPluginDefinition(ctx context.Context, name, namespace string, opts ...func(definition *greenhousev1alpha1.PluginDefinition)) *greenhousev1alpha1.PluginDefinition {
	pd := &greenhousev1alpha1.PluginDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
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
	return pd
}

// WithClusterPluginDefinition sets the PluginDefinition reference to ClusterPluginDefinition in the Plugin
func WithClusterPluginDefinition(pluginDefinition string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		p.Spec.PluginDefinitionRef = greenhousev1alpha1.PluginDefinitionReference{
			Name: pluginDefinition,
			Kind: greenhousev1alpha1.ClusterPluginDefinitionKind,
		}
	}
}

// WithPluginDefinition sets the PluginDefinition reference to namespaced PluginDefinition in the Plugin
func WithPluginDefinition(pluginDefinition string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		p.Spec.PluginDefinitionRef = greenhousev1alpha1.PluginDefinitionReference{
			Name: pluginDefinition,
			Kind: greenhousev1alpha1.PluginDefinitionKind,
		}
	}
}

// WithReleaseNamespace sets the ReleaseNamespace of a Plugin
func WithReleaseNamespace(releaseNamespace string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		p.Spec.ReleaseNamespace = releaseNamespace
	}
}

// WithReleaseName sets the ReleaseName of a Plugin
func WithReleaseName(releaseName string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		p.Spec.ReleaseName = releaseName
	}
}

// WithCluster sets the Cluster for a Plugin
func WithCluster(cluster string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		p.Spec.ClusterName = cluster
	}
}

// WithPresetLabelValue sets the value of the greenhouseapis.LabelKeyPluginPreset label on a Plugin
// This label is used to indicate that the Plugin is managed by a PluginPreset.
func WithPresetLabelValue(value string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		if p.Labels == nil {
			p.Labels = make(map[string]string, 1)
		}
		p.Labels[greenhouseapis.LabelKeyPluginPreset] = value
	}
}

// WithPluginLabel sets the label on a Plugin
func WithPluginLabel(key, value string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		if p.Labels == nil {
			p.Labels = make(map[string]string, 1)
		}
		p.Labels[key] = value
	}
}

// WithPluginOptionValue sets the value of a PluginOptionValue, clears ValueFrom and Template
func WithPluginOptionValue(name string, value *apiextensionsv1.JSON) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		for i, v := range p.Spec.OptionValues {
			if v.Name == name {
				v.Value = value
				v.ValueFrom = nil
				v.Template = nil
				p.Spec.OptionValues[i] = v
				return
			}
		}
		p.Spec.OptionValues = append(p.Spec.OptionValues, greenhousev1alpha1.PluginOptionValue{
			Name:      name,
			Value:     value,
			ValueFrom: nil,
			Template:  nil,
		})
	}
}

// WithPluginOptionValue sets the value of a PluginOptionValue, clears Value and Template
func WithPluginOptionValueFrom(name string, valueFrom *greenhousev1alpha1.ValueFromSource) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		for i, v := range p.Spec.OptionValues {
			if v.Name == name {
				v.Value = nil
				v.ValueFrom = valueFrom
				v.Template = nil
				p.Spec.OptionValues[i] = v
				return
			}
		}
		p.Spec.OptionValues = append(p.Spec.OptionValues, greenhousev1alpha1.PluginOptionValue{
			Name:      name,
			Value:     nil,
			ValueFrom: valueFrom,
			Template:  nil,
		})
	}
}

// WithPluginOptionValue sets the template of a PluginOptionValue,
func WithPluginOptionValueTemplate(name string, template *string) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		for i, v := range p.Spec.OptionValues {
			if v.Name == name {
				v.Value = nil
				v.Template = template
				v.ValueFrom = nil
				p.Spec.OptionValues[i] = v
				return
			}
		}
		p.Spec.OptionValues = append(p.Spec.OptionValues, greenhousev1alpha1.PluginOptionValue{
			Name:      name,
			Value:     nil,
			Template:  template,
			ValueFrom: nil,
		})
	}
}

func WithPluginWaitFor(waitFor []greenhousev1alpha1.WaitForItem) func(*greenhousev1alpha1.Plugin) {
	return func(p *greenhousev1alpha1.Plugin) {
		p.Spec.WaitFor = waitFor
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

// NewPlugin returns a greenhousev1alpha1.Plugin object. Opts can be used to set the desired state of the Plugin.
func NewPlugin(ctx context.Context, name, namespace string, opts ...func(*greenhousev1alpha1.Plugin)) *greenhousev1alpha1.Plugin {
	GinkgoHelper()
	plugin := &greenhousev1alpha1.Plugin{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Plugin",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	for _, o := range opts {
		o(plugin)
	}
	return plugin
}

// WithPluginPresetClusterSelector sets the ClusterSelector on a PluginPreset.
func WithPluginPresetClusterSelector(clusterSelector metav1.LabelSelector) func(*greenhousev1alpha1.PluginPreset) {
	return func(pp *greenhousev1alpha1.PluginPreset) {
		pp.Spec.ClusterSelector = clusterSelector
	}
}

// WithPluginPresetPluginSpec sets the applicable fields from a PluginSpec on the PluginPreset.
func WithPluginSpec(pluginSpec greenhousev1alpha1.PluginSpec) func(*greenhousev1alpha1.PluginPreset) {
	return func(pp *greenhousev1alpha1.PluginPreset) {
		pp.Spec.Plugin.PluginDefinition = pluginSpec.PluginDefinition //nolint:staticcheck
		pp.Spec.Plugin.PluginDefinitionRef = pluginSpec.PluginDefinitionRef
		pp.Spec.Plugin.DisplayName = pluginSpec.DisplayName
		pp.Spec.Plugin.OptionValues = pluginSpec.OptionValues
		pp.Spec.Plugin.ReleaseNamespace = pluginSpec.ReleaseNamespace
		pp.Spec.Plugin.ReleaseName = pluginSpec.ReleaseName
	}
}

// WithPluginPresetPluginSpec sets the PluginSpec on a PluginPreset.
func WithPluginPresetPluginSpec(pluginSpec greenhousev1alpha1.PluginPresetPluginSpec) func(*greenhousev1alpha1.PluginPreset) {
	return func(pp *greenhousev1alpha1.PluginPreset) {
		pp.Spec.Plugin = pluginSpec
	}
}

// WithPluginPresetLabel sets the label on a PluginPreset
func WithPluginPresetLabel(key, value string) func(*greenhousev1alpha1.PluginPreset) {
	return func(pp *greenhousev1alpha1.PluginPreset) {
		if pp.Labels == nil {
			pp.Labels = make(map[string]string, 1)
		}
		pp.Labels[key] = value
	}
}

// WithPluginPresetAnnotation sets the annotation on a PluginPreset
func WithPluginPresetAnnotation(key, value string) func(*greenhousev1alpha1.PluginPreset) {
	return func(pp *greenhousev1alpha1.PluginPreset) {
		if pp.Annotations == nil {
			pp.Annotations = make(map[string]string, 1)
		}
		pp.Annotations[key] = value
	}
}

// WithClusterOverrides sets the ClusterOverrides for a Cluster
func WithClusterOverride(clusterName string, optionValues []greenhousev1alpha1.PluginOptionValue) func(*greenhousev1alpha1.PluginPreset) {
	return func(pp *greenhousev1alpha1.PluginPreset) {
		for co := range pp.Spec.ClusterOptionOverrides {
			if pp.Spec.ClusterOptionOverrides[co].ClusterName == clusterName {
				pp.Spec.ClusterOptionOverrides[co].Overrides = optionValues
				return
			}
		}
		pp.Spec.ClusterOptionOverrides = append(pp.Spec.ClusterOptionOverrides, greenhousev1alpha1.ClusterOptionOverride{
			ClusterName: clusterName,
			Overrides:   optionValues,
		})
	}
}

// WithPluginPresetDeletionPolicy sets the DeletionPolicy on a PluginPreset.
func WithPluginPresetDeletionPolicy(deletionPolicy string) func(*greenhousev1alpha1.PluginPreset) {
	return func(pp *greenhousev1alpha1.PluginPreset) {
		pp.Spec.DeletionPolicy = deletionPolicy
	}
}

// WithPluginPresetWaitFor sets the WaitFor on a PluginPreset.
func WithPluginPresetWaitFor(waitFor ...greenhousev1alpha1.WaitForItem) func(*greenhousev1alpha1.PluginPreset) {
	return func(pp *greenhousev1alpha1.PluginPreset) {
		pp.Spec.WaitFor = waitFor
	}
}

// NewPluginPreset returns a greenhousev1alpha1.PluginPreset object. Opts can be used to set the desired state of the PluginPreset.
func NewPluginPreset(name, namespace string, opts ...func(*greenhousev1alpha1.PluginPreset)) *greenhousev1alpha1.PluginPreset {
	GinkgoHelper()
	pluginPreset := &greenhousev1alpha1.PluginPreset{
		TypeMeta: metav1.TypeMeta{
			Kind:       greenhousev1alpha1.PluginPresetKind,
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	for _, o := range opts {
		o(pluginPreset)
	}
	return pluginPreset
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

// NewTeamRole returns a greenhousev1alpha1.TeamRole object. Opts can be used to set the desired state of the TeamRole.
func NewTeamRole(ctx context.Context, name, namespace string, opts ...func(*greenhousev1alpha1.TeamRole)) *greenhousev1alpha1.TeamRole {
	tr := &greenhousev1alpha1.TeamRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TeamRole",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
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

func WithTeamRoleRef(roleRef string) func(*greenhousev1alpha2.TeamRoleBinding) {
	return func(trb *greenhousev1alpha2.TeamRoleBinding) {
		trb.Spec.TeamRoleRef = roleRef
	}
}

func WithTeamRef(teamRef string) func(*greenhousev1alpha2.TeamRoleBinding) {
	return func(trb *greenhousev1alpha2.TeamRoleBinding) {
		trb.Spec.TeamRef = teamRef
	}
}

func WithClusterName(clusterName string) func(*greenhousev1alpha2.TeamRoleBinding) {
	return func(trb *greenhousev1alpha2.TeamRoleBinding) {
		trb.Spec.ClusterSelector = greenhousev1alpha2.ClusterSelector{
			Name: clusterName,
		}
	}
}

func WithClusterSelector(selector metav1.LabelSelector) func(*greenhousev1alpha2.TeamRoleBinding) {
	return func(trb *greenhousev1alpha2.TeamRoleBinding) {
		trb.Spec.ClusterSelector.LabelSelector = selector
	}
}

func WithNamespaces(namespaces ...string) func(*greenhousev1alpha2.TeamRoleBinding) {
	return func(trb *greenhousev1alpha2.TeamRoleBinding) {
		trb.Spec.Namespaces = namespaces
	}
}

func WithCreateNamespace(createNamespaces bool) func(*greenhousev1alpha2.TeamRoleBinding) {
	return func(trb *greenhousev1alpha2.TeamRoleBinding) {
		trb.Spec.CreateNamespaces = createNamespaces
	}
}

func WithUsernames(usernames []string) func(*greenhousev1alpha2.TeamRoleBinding) {
	return func(trb *greenhousev1alpha2.TeamRoleBinding) {
		trb.Spec.Usernames = usernames
	}
}

// WithTeamRoleBindingLabel sets the label on a TeamRoleBinding
func WithTeamRoleBindingLabel(key, value string) func(*greenhousev1alpha2.TeamRoleBinding) {
	return func(trb *greenhousev1alpha2.TeamRoleBinding) {
		if trb.Labels == nil {
			trb.Labels = make(map[string]string, 1)
		}
		trb.Labels[key] = value
	}
}

// NewTeamRoleBinding returns a greenhousev1alpha2.TeamRoleBinding object. Opts can be used to set the desired state of the TeamRoleBinding.
func NewTeamRoleBinding(ctx context.Context, name, namespace string, opts ...func(*greenhousev1alpha2.TeamRoleBinding)) *greenhousev1alpha2.TeamRoleBinding {
	trb := &greenhousev1alpha2.TeamRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TeamRoleBinding",
			APIVersion: greenhousev1alpha2.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	for _, o := range opts {
		o(trb)
	}
	return trb
}

func WithMappedIDPGroup(group string) func(*greenhousev1alpha1.Team) {
	return func(t *greenhousev1alpha1.Team) {
		t.Spec.MappedIDPGroup = group
	}
}

// WithTeamLabel sets the label on a Team
func WithTeamLabel(key, value string) func(*greenhousev1alpha1.Team) {
	return func(t *greenhousev1alpha1.Team) {
		if t.Labels == nil {
			t.Labels = make(map[string]string, 1)
		}
		t.Labels[key] = value
	}
}

// NewTeam returns a greenhousev1alpha1.Team object. Opts can be used to set the desired state of the Team.
func NewTeam(ctx context.Context, name, namespace string, opts ...func(*greenhousev1alpha1.Team)) *greenhousev1alpha1.Team {
	team := &greenhousev1alpha1.Team{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Team",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	for _, opt := range opts {
		opt(team)
	}
	return team
}

// WithSecretType sets the type of the Secret
func WithSecretType(secretType corev1.SecretType) func(*corev1.Secret) {
	return func(s *corev1.Secret) {
		s.Type = secretType
	}
}

func WithSecretAnnotations(annotations map[string]string) func(*corev1.Secret) {
	return func(s *corev1.Secret) {
		s.SetAnnotations(annotations)
	}
}

func WithSecretLabels(labels map[string]string) func(*corev1.Secret) {
	return func(s *corev1.Secret) {
		s.SetLabels(labels)
	}
}

// WithSecretLabel sets the label on a Secret
func WithSecretLabel(key, value string) func(*corev1.Secret) {
	return func(s *corev1.Secret) {
		if s.Labels == nil {
			s.Labels = make(map[string]string, 1)
		}
		s.Labels[key] = value
	}
}

// WithSecretData sets the data of the Secret
func WithSecretData(data map[string][]byte) func(*corev1.Secret) {
	return func(s *corev1.Secret) {
		s.Data = data
	}
}

// WithSecretNamespace sets the namespace of the Secret
func WithSecretNamespace(namespace string) func(*corev1.Secret) {
	return func(s *corev1.Secret) {
		s.Namespace = namespace
	}
}

// NewSecret returns a Secret object. Opts can be used to set the desired state of the Secret.
func NewSecret(name, namespace string, opts ...func(*corev1.Secret)) *corev1.Secret {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.GroupName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	for _, opt := range opts {
		opt(secret)
	}
	return secret
}

func WithConfigMapLabels(labels map[string]string) func(*corev1.ConfigMap) {
	return func(cm *corev1.ConfigMap) {
		cm.SetLabels(labels)
	}
}

func WithConfigMapData(data map[string]string) func(*corev1.ConfigMap) {
	return func(cm *corev1.ConfigMap) {
		cm.Data = data
	}
}

func NewConfigMap(name, namespace string, opts ...func(*corev1.ConfigMap)) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	return cm
}

func WithRepository(url string) func(source *greenhousev1alpha1.CatalogSource) {
	return func(source *greenhousev1alpha1.CatalogSource) {
		source.Repository = url
	}
}

func WithRepositoryBranch(branch string) func(source *greenhousev1alpha1.CatalogSource) {
	return func(source *greenhousev1alpha1.CatalogSource) {
		if source.Ref == nil {
			source.Ref = &greenhousev1alpha1.GitRef{}
		}
		source.Ref.Branch = ptr.To(branch)
	}
}

func WithRepositoryTag(tag string) func(source *greenhousev1alpha1.CatalogSource) {
	return func(source *greenhousev1alpha1.CatalogSource) {
		if source.Ref == nil {
			source.Ref = &greenhousev1alpha1.GitRef{}
		}
		source.Ref.Tag = ptr.To(tag)
	}
}

func WithRepositorySHA(sha string) func(source *greenhousev1alpha1.CatalogSource) {
	return func(source *greenhousev1alpha1.CatalogSource) {
		if source.Ref == nil {
			source.Ref = &greenhousev1alpha1.GitRef{}
		}
		source.Ref.SHA = ptr.To(sha)
	}
}

func WithOverrides(overrides []greenhousev1alpha1.CatalogOverrides) func(source *greenhousev1alpha1.CatalogSource) {
	return func(source *greenhousev1alpha1.CatalogSource) {
		if len(source.Overrides) == 0 {
			source.Overrides = make([]greenhousev1alpha1.CatalogOverrides, 0, len(overrides))
		}
		source.Overrides = append(source.Overrides, overrides...)
	}
}

func WithCatalogResources(resources []string) func(source *greenhousev1alpha1.CatalogSource) {
	return func(source *greenhousev1alpha1.CatalogSource) {
		if len(source.Resources) == 0 {
			source.Resources = make([]string, 0, len(resources))
		}
		source.Resources = append(source.Resources, resources...)
	}
}

func NewCatalogSource(opts ...func(source *greenhousev1alpha1.CatalogSource)) greenhousev1alpha1.CatalogSource {
	source := &greenhousev1alpha1.CatalogSource{}
	for _, o := range opts {
		o(source)
	}
	return *source
}

func NewCatalog(name, namespace string, sources ...greenhousev1alpha1.CatalogSource) *greenhousev1alpha1.Catalog {
	catalog := &greenhousev1alpha1.Catalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: greenhousev1alpha1.CatalogSpec{
			Sources: []greenhousev1alpha1.CatalogSource{},
		},
	}
	catalog.Spec.Sources = append(catalog.Spec.Sources, sources...)
	return catalog
}

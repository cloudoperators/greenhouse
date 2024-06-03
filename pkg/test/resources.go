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

// OnboardCluster creates a new Cluster and Kubernetes secret for a remote cluster and creates the namespace used for TestSetup on the remote cluster
func (t *TestSetup) OnboardCluster(ctx context.Context, name string, kubeCfg []byte) *greenhousev1alpha1.Cluster {
	GinkgoHelper()
	clusterName := t.RandomizeName(name)
	cluster := &greenhousev1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: greenhousev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: t.Namespace(),
		},
		Spec: greenhousev1alpha1.ClusterSpec{
			AccessMode: greenhousev1alpha1.ClusterAccessModeDirect,
		},
	}
	Expect(t.Create(ctx, cluster)).To(Succeed(), "there should be no error creating the cluster during onboarding")

	var testClusterK8sSecret = &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.GroupName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
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

// WithRules overrides the default rules of a TeamRole
func WithRules(rules []rbacv1.PolicyRule) func(*greenhousev1alpha1.TeamRole) {
	return func(tr *greenhousev1alpha1.TeamRole) {
		tr.Spec.Rules = rules
	}
}

// CreateTeamRole returns a TeamRole object. Opts can be used to set the desired state of the TeamRole.
func (t *TestSetup) CreateTeamRole(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.TeamRole)) *greenhousev1alpha1.TeamRole {
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

func WithNamespaces(namespaces ...string) func(*greenhousev1alpha1.TeamRoleBinding) {
	return func(trb *greenhousev1alpha1.TeamRoleBinding) {
		trb.Spec.Namespaces = namespaces
	}
}

// CreateTeamRoleBinding returns a TeamRoleBinding object. Opts can be used to set the desired state of the TeamRoleBinding.
func (t *TestSetup) CreateTeamRoleBinding(ctx context.Context, name string, opts ...func(*greenhousev1alpha1.TeamRoleBinding)) *greenhousev1alpha1.TeamRoleBinding {
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

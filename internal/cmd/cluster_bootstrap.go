// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	greenhouseapis "github.com/cloudoperators/greenhouse/api"
	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/common"
	clustercontroller "github.com/cloudoperators/greenhouse/internal/controller/cluster/utils"
)

const (
	serviceAccountName = "greenhouse"
)

var (
	setupLog                 = ctrl.Log.WithName("setup")
	clusterBootstrapCmdUsage = "bootstrap"
	ctx                      = context.Background()
)

type newClusterBootstrapOptions struct {
	customerClient       client.Client
	customerConfig       rest.Config
	ghClient             client.Client
	ghConfig             rest.Config
	kubecontext          string
	kubeconfig           string
	orgName              string
	clusterName          string
	greenhouseKubeConfig string
	onBehafOfUser        string
}

func init() {
	clusterCmd.AddCommand(newClusterBootstrapCmd())
}

func newClusterBootstrapCmd() *cobra.Command {
	o := &newClusterBootstrapOptions{}
	bootstrapCmd := &cobra.Command{
		Use:   clusterBootstrapCmdUsage,
		Short: "Bootstrap a Kubernetes cluster to Greenhouse",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.fillDefaults(); err != nil {
				return err
			}
			return o.run()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.ValidateRequiredFlags(); err != nil {
				return err
			}
			o.kubeconfig = cmd.Flag("kubeconfig").Value.String()
			return o.permissionCheck()
		},
	}

	// Flags
	bootstrapCmd.Flags().AddGoFlagSet(flag.CommandLine)

	bootstrapCmd.Flags().StringVar(&o.kubecontext, "kubecontext", "", "The context to use from the kubeconfig for the cluster which needs to be onboarded(defaults to current-context)")
	bootstrapCmd.Flags().StringVar(&o.orgName, "org", clientutil.GetEnvOrDefault("GREENHOUSE_ORG", ""), "The organization name to use. Can be set via GREENHOUSE_ORG env var")
	bootstrapCmd.Flags().StringVar(&o.clusterName, "cluster-name", clientutil.GetEnvOrDefault("GREENHOUSE_CLUSTER_NAME", ""), "The cluster name to use. Can be set via GREENHOUSE_CLUSTER_NAME env var")
	bootstrapCmd.Flags().StringVar(&o.greenhouseKubeConfig, "greenhouse-kubeconfig", "", "The kubeconfig of the greenhouse cluster")
	bootstrapCmd.Flags().StringVar(&o.onBehafOfUser, "as", "", "The user to impersonate for the operation")

	// Mark required flags
	if err := bootstrapCmd.MarkFlagRequired("org"); err != nil {
		setupLog.Error(err, "Flag could not set as required", "org")
	}
	if err := bootstrapCmd.MarkFlagRequired("bootstrap-kubeconfig"); err != nil {
		setupLog.Error(err, "Flag could not set as required", "bootstrap-kubeconfig")
	}
	bootstrapCmd.MarkFlagsRequiredTogether("org", "greenhouse-kubeconfig")
	// Silence usage to avoid confusing customers
	bootstrapCmd.SilenceUsage = true

	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.ISO8601TimeEncoder,
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	return bootstrapCmd
}

func (o *newClusterBootstrapOptions) permissionCheck() (err error) {
	greenhouseRestConfig := getClientKubeconfig(&o.greenhouseKubeConfig)
	o.ghConfig = *greenhouseRestConfig
	o.ghClient, err = clientutil.NewK8sClient(&o.ghConfig)
	if err != nil {
		return err
	}
	customerRestConfig := getKubeconfigOrDie(o.kubecontext)
	o.customerConfig = *customerRestConfig
	o.customerClient, err = clientutil.NewK8sClient(&o.customerConfig)
	if err != nil {
		return err
	}

	clientErr := o.customerClient.Get(ctx, client.ObjectKey{Name: corev1.NamespaceDefault}, &corev1.Namespace{})
	if clientErr != nil {
		setupLog.Info("Missing permissions: getNamespace", "clusterName", o.customerConfig.Host)
		setupLog.Error(clientErr, "", "clusterName", o.customerConfig.Host)
		return clientErr
	}
	greenhouseMissingPermissions := common.CheckGreenhousePermission(ctx, o.ghClient, o.onBehafOfUser, o.orgName)
	if len(greenhouseMissingPermissions) > 0 {
		setupLog.Info("Missing permissions: "+fmt.Sprintf("%v", greenhouseMissingPermissions)+" clusterName", o.ghConfig.Host)
	}

	clientMissingPermissions := common.CheckClientClusterPermission(ctx, o.customerClient, o.onBehafOfUser, corev1.NamespaceDefault)
	if len(clientMissingPermissions) > 0 {
		setupLog.Info("Missing permissions: "+fmt.Sprintf("%v", clientMissingPermissions)+" clusterName", o.customerConfig.Host)
	}

	if clientMissingPermissions != nil || greenhouseMissingPermissions != nil {
		return errors.New("missing one of more permissions, please request them before retrying")
	}

	return nil
}

func (o *newClusterBootstrapOptions) fillDefaults() error {
	var clientKubeConfig string
	if o.kubeconfig == "" {
		clientKubeConfig = os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	} else {
		clientKubeConfig = o.kubeconfig
	}
	if o.clusterName == "" {
		customerConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: clientKubeConfig},
			&clientcmd.ConfigOverrides{
				CurrentContext: o.kubecontext,
			}).RawConfig()
		if err != nil {
			return err
		}
		o.clusterName = customerConfig.CurrentContext
	}
	// Validate the cluster name here before proceeding
	if err := validateClusterName(o.clusterName, 40); err != nil {
		return err
	}

	return nil
}

func (o *newClusterBootstrapOptions) run() error {
	bootstrapped := o.isClusterAlreadyBootstraped(ctx)
	if !bootstrapped {
		setupLog.Info("Bootstraping cluster", "clusterName", o.clusterName, "orgName", o.orgName)
	} else {
		setupLog.Info("Cluster exists, checking resources for update", "clusterName", o.clusterName, "orgName", o.orgName)
	}
	if err := createNameSpaceInRemoteCluster(ctx, o.customerClient, o.orgName); err != nil {
		return err
	}
	if err := createServiceAccountInRemoteCluster(ctx, o.customerClient, serviceAccountName, o.orgName); err != nil {
		return err
	}
	if err := createClusterRoleBindingInRemoteCluster(ctx, o.customerClient, o.orgName); err != nil {
		return err
	}

	if err := o.createOrUpdateClusterObject(ctx, bootstrapped); err != nil {
		return err
	}

	setupLog.Info("Bootstraping cluster finished", "clusterName", o.clusterName, "orgName", o.orgName)
	return nil
}

func createNameSpaceInRemoteCluster(ctx context.Context, k8sClient client.Client, orgName string) error {
	var namespace = new(corev1.Namespace)
	namespace.Name = orgName
	result, err := clientutil.CreateOrPatch(ctx, k8sClient, namespace, func() error {
		return nil
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultNone:
		setupLog.Info("namespace already exists", "name", namespace.Name)
	case clientutil.OperationResultCreated:
		setupLog.Info("created namespace", "name", namespace.Name)
	case clientutil.OperationResultUpdated:
		setupLog.Info("updated namespace", "name", namespace.Name)
	}
	return nil
}

func createServiceAccountInRemoteCluster(ctx context.Context, k8sClient client.Client, svAccName, orgName string) error {
	var serviceAccount = new(corev1.ServiceAccount)
	serviceAccount.Name = svAccName
	serviceAccount.Namespace = orgName
	result, err := clientutil.CreateOrPatch(ctx, k8sClient, serviceAccount, func() error {
		return nil
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultNone:
		setupLog.Info("serviceAccount already exists", "name", serviceAccount.Name)
	case clientutil.OperationResultCreated:
		setupLog.Info("created serviceAccount", "name", serviceAccount.Name)
	case clientutil.OperationResultUpdated:
		setupLog.Info("updated serviceAccount", "name", serviceAccount.Name)
	}

	return nil
}

func createClusterRoleBindingInRemoteCluster(ctx context.Context, k8sClient client.Client, orgName string) error {
	var clusterRoleBinding = new(rbacv1.ClusterRoleBinding)
	clusterRoleBinding.Name = serviceAccountName

	var namespace = new(corev1.Namespace)
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "", Name: orgName}, namespace); err != nil {
		return err
	}

	result, err := clientutil.CreateOrPatch(ctx, k8sClient, clusterRoleBinding, func() error {
		clusterRoleBinding.Subjects = []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      serviceAccountName,
				Namespace: orgName,
			},
		}
		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
			APIGroup: rbacv1.GroupName,
		}
		return controllerutil.SetOwnerReference(namespace, clusterRoleBinding, k8sClient.Scheme())
	})

	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultNone:
		setupLog.Info("clusterRoleBinding already exists", "name", clusterRoleBinding.Name)
	case clientutil.OperationResultCreated:
		setupLog.Info("created clusterRoleBinding", "name", clusterRoleBinding.Name)
	case clientutil.OperationResultUpdated:
		setupLog.Info("updated clusterRoleBinding", "name", clusterRoleBinding.Name)
	}
	return nil
}

func (o *newClusterBootstrapOptions) isClusterAlreadyBootstraped(ctx context.Context) bool {
	var cluster = new(greenhouseapisv1alpha1.Cluster)
	if err := o.ghClient.Get(ctx, client.ObjectKey{Name: o.clusterName, Namespace: o.orgName}, cluster); err != nil {
		return false
	}
	return true
}

func (o *newClusterBootstrapOptions) createOrUpdateClusterObject(ctx context.Context, bootstrapped bool) error {
	token, err := o.createServiceAccountToken()
	if err != nil {
		return err
	}

	generateKubeconfig := &clustercontroller.KubeConfigHelper{
		Host:          o.customerConfig.Host,
		TLSServerName: o.customerConfig.ServerName,
		CAData:        o.customerConfig.CAData,
		BearerToken:   token,
		Username:      serviceAccountName,
		Namespace:     o.orgName,
	}

	genKubeConfig, err := clientcmd.Write(generateKubeconfig.RestConfigToAPIConfig(o.clusterName))
	if err != nil {
		return err
	}
	clusterSecret := new(corev1.Secret)
	clusterSecret.Name = o.clusterName
	clusterSecret.Namespace = o.orgName
	clusterSecret.Type = greenhouseapis.SecretTypeKubeConfig
	if !bootstrapped {
		clusterSecret.Data = map[string][]byte{greenhouseapis.KubeConfigKey: genKubeConfig}
		if err = o.ghClient.Create(ctx, clusterSecret); err != nil {
			return err
		}
		setupLog.Info("created clusterSecret", "name", clusterSecret.Name)
	} else {
		clusterSecret.Data = map[string][]byte{greenhouseapis.KubeConfigKey: genKubeConfig}
		clusterSecret.Data = map[string][]byte{greenhouseapis.GreenHouseKubeConfigKey: genKubeConfig}
		if err = o.ghClient.Update(ctx, clusterSecret); err != nil {
			return err
		}
		setupLog.Info("updated clusterSecret", "name", clusterSecret.Name)
	}

	return nil
}

func (o *newClusterBootstrapOptions) createServiceAccountToken() (token string, err error) {
	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: ptr.To(int64(1 * time.Hour / time.Second)),
		},
	}
	clientset, err := kubernetes.NewForConfig(&o.customerConfig)
	if err != nil {
		return "", err
	}
	tokenRequestResponse, err := clientset.
		CoreV1().
		ServiceAccounts(o.orgName).
		CreateToken(ctx, serviceAccountName, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	return tokenRequestResponse.Status.Token, nil
}

func getClientKubeconfig(kubeconfig *string) *rest.Config {
	restConfig, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		setupLog.Error(err, "Failed to load kubeconfig")
		os.Exit(1)
	}
	setupLog.Info("Loaded client kubeconfig", "host", restConfig.Host)
	return restConfig
}

func getKubeconfigOrDie(kubecontext string) *rest.Config {
	if kubecontext == "" {
		kubecontext = os.Getenv("KUBECONTEXT")
	}
	restConfig, err := config.GetConfigWithContext(kubecontext)
	if err != nil {
		setupLog.Error(err, "Failed to load kubeconfig")
		os.Exit(1)
	}
	setupLog.Info("Loaded kubeconfig", "context", kubecontext, "host", restConfig.Host)
	return restConfig
}

// validateClusterName is making sure that the cluster name is not empty, not longer than length parameter characters and does not contain '--'
func validateClusterName(clusterName string, length int) error {
	switch {
	case clusterName == "":
		return errors.New("cluster name cannot be empty")
	case len(clusterName) > length:
		return errors.New("cluster name cannot be longer than 40 characters")
	case strings.Contains(clusterName, "--"):
		return errors.New("cluster name cannot contain '--'")
	}
	return nil
}

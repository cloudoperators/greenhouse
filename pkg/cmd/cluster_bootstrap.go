// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"errors"
	"flag"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
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

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhouseapisv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	clustercontroller "github.com/cloudoperators/greenhouse/pkg/controllers/cluster"
)

const (
	tailscaleServiceAccountName = "tailscale"
	serviceAccountName          = "greenhouse"
)

var (
	setupLog                 = ctrl.Log.WithName("setup")
	clusterBootstrapCmdUsage = "bootstrap"
	ctx                      = context.Background()
	// Permission map for the greenhouse cluster
	greenhousePermission = map[string][]string{
		"createCluster": {"create", "greenhouse.sap", "clusters"},
		"deleteCluster": {"delete", "greenhouse.sap", "clusters"},
		"updateCluster": {"update", "greenhouse.sap", "clusters"},
		"patchCluster":  {"patch", "greenhouse.sap", "clusters"},
		"createSecret":  {"create", "", "secrets"},
		"updateSecret":  {"update", "", "secrets"},
		"patchSecret":   {"patch", "", "secrets"},
	}
	// Permission map for the customer cluster
	clientClusterPermission = map[string][]string{
		"clusterAdmin": {"*", "*", "*"},
	}
)

type newClusterBootstrapOptions struct {
	customerClient     client.Client
	customerConfig     rest.Config
	ghClient           client.Client
	ghConfig           rest.Config
	kubecontext        string
	headscaleURI       string
	preAuthKey         string
	orgName            string
	clusterName        string
	proxyImage         string
	proxyImageTag      string
	tailscaleImage     string
	tailscaleImageTag  string
	customerKubeConfig string
	onBehafOfUser      string
	freeAccessMode     bool
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
			return o.permissionCheck()
		},
	}

	// Flags
	bootstrapCmd.Flags().AddGoFlagSet(flag.CommandLine)

	bootstrapCmd.Flags().StringVar(&o.kubecontext, "kubecontext", "", "The context to use from the kubeconfig (defaults to current-context)")
	bootstrapCmd.Flags().StringVar(&o.headscaleURI, "headscale-uri", clientutil.GetEnvOrDefault("HEADSCALE_URI", ""), "The headscale URI to use. Can be set via HEADSCALE_URI env var")
	bootstrapCmd.Flags().StringVar(&o.preAuthKey, "preauth-key", clientutil.GetEnvOrDefault("HEADSCALE_PREAUTHKEY", ""), "The pre-auth key to use. Can be set via HEADSCALE_PREAUTHKEY env var")
	bootstrapCmd.Flags().StringVar(&o.orgName, "org", clientutil.GetEnvOrDefault("GREENHOUSE_ORG", ""), "The organization name to use. Can be set via GREENHOUSE_ORG env var")
	bootstrapCmd.Flags().StringVar(&o.clusterName, "cluster-name", clientutil.GetEnvOrDefault("GREENHOUSE_CLUSTER_NAME", ""), "The cluster name to use. Can be set via GREENHOUSE_CLUSTER_NAME env var")
	bootstrapCmd.Flags().StringVar(&o.proxyImage, "proxy-image", clientutil.GetEnvOrDefault("PROXY_IMAGE", "ghcr.io/cloudoperators/tcp-proxy"), "The proxy image to use. Can be set via PROXY_IMAGE env var")
	bootstrapCmd.Flags().StringVar(&o.proxyImageTag, "proxy-image-tag", clientutil.GetEnvOrDefault("PROXY_IMAGE_TAG", "0.1.1"), "The proxy image tag to use. Can be set via PROXY_IMAGE_TAG env var")
	bootstrapCmd.Flags().StringVar(&o.tailscaleImage, "tailscale-image", clientutil.GetEnvOrDefault("TAILSCALE_IMAGE", "ghcr.io/cloudoperators/tailscale"), "The tailscale image to use. Can be set via TAILSCALE_IMAGE env var")
	bootstrapCmd.Flags().StringVar(&o.tailscaleImageTag, "tailscale-image-tag", clientutil.GetEnvOrDefault("TAILSCALE_IMAGE_TAG", "1.50.1"), "The tailscale image tag to use. Can be set via TAILSCALE_IMAGE_TAG env var")
	bootstrapCmd.Flags().StringVar(&o.customerKubeConfig, "bootstrap-kubeconfig", "", "The kubeconfig of the cluster to bootstrap")
	bootstrapCmd.Flags().BoolVar(&o.freeAccessMode, "free-access-mode", true, "Let the cluster bootstrap controller decide the access mode for the cluster")
	bootstrapCmd.Flags().StringVar(&o.onBehafOfUser, "as", "", "The user to impersonate for the operation")

	// Mark required flags
	if err := bootstrapCmd.MarkFlagRequired("org"); err != nil {
		setupLog.Error(err, "Flag could not set as required", "org")
	}
	if err := bootstrapCmd.MarkFlagRequired("bootstrap-kubeconfig"); err != nil {
		setupLog.Error(err, "Flag could not set as required", "bootstrap-kubeconfig")
	}
	bootstrapCmd.MarkFlagsRequiredTogether("org", "bootstrap-kubeconfig")
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
	greenhouseRestConfig := getKubeconfigOrDie(o.kubecontext)
	o.ghConfig = *greenhouseRestConfig
	o.ghClient, err = clientutil.NewK8sClient(&o.ghConfig)
	if err != nil {
		return err
	}
	customerRestConfig := getClientKubeconfig(&o.customerKubeConfig)
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
	greenhouseMissingPermissions := checkPermissionMap(o.ghClient, greenhousePermission, o.onBehafOfUser, o.orgName)
	if len(greenhouseMissingPermissions) > 0 {
		setupLog.Info("Missing permissions: "+strings.Join(greenhouseMissingPermissions, ","), "clusterName", o.ghConfig.Host)
	}

	clientMissingPermissions := checkPermissionMap(o.customerClient, clientClusterPermission, o.onBehafOfUser, corev1.NamespaceDefault)
	if len(clientMissingPermissions) > 0 {
		setupLog.Info("Missing permissions: "+strings.Join(clientMissingPermissions, ","), "clusterName", o.customerConfig.Host)
	}

	if clientMissingPermissions != nil || greenhouseMissingPermissions != nil {
		return errors.New("missing one of more permissions, please request them before retrying")
	}

	return nil
}

func (o *newClusterBootstrapOptions) fillDefaults() error {
	if o.clusterName == "" {
		customerConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: o.customerKubeConfig},
			&clientcmd.ConfigOverrides{
				CurrentContext: "",
			}).RawConfig()
		if err != nil {
			return err
		}
		o.clusterName = customerConfig.CurrentContext
	}

	// This is temporary solution till we have a headscale per org deployed
	switch {
	case strings.Contains(o.ghConfig.Host, "api.qa.greenhouse"):
		o.headscaleURI = "https://headscale.greenhouse-qa.eu-nl-1.cloud.sap"
	case strings.Contains(o.ghConfig.Host, "api.greenhouse-qa"):
		o.headscaleURI = "https://headscale.greenhouse-qa.eu-nl-1.cloud.sap"
	case strings.Contains(o.ghConfig.Host, "api.dev-david.greenhouse"):
		o.headscaleURI = "https://headscale.davidg.c.eu-nl-1.cloud.sap"
	default:
		o.headscaleURI = "https://headscale.greenhouse.global.cloud.sap"
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
	if err := createServiceAccountInRemoteCluster(ctx, o.customerClient, tailscaleServiceAccountName, o.orgName); err != nil {
		return err
	}
	if err := createRoleInRemoteCluster(ctx, o.customerClient, o.orgName); err != nil {
		return err
	}
	if err := createRoleBindingInRemoteCluster(ctx, o.customerClient, o.orgName); err != nil {
		return err
	}
	if !bootstrapped {
		if err := o.createClusterObject(ctx); err != nil {
			return err
		}
	}
	/*
		else {
			// TODO: if the cluster already bootstraped we need to check if the preauthkey is still correct
			// New bootstraptoken needs to be created and the cluster secret object needs to be updated
		}
	*/
	if err := o.workOnCreatedCluster(ctx); err != nil {
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

func createRoleInRemoteCluster(ctx context.Context, k8sClient client.Client, orgName string) error {
	var role = new(rbacv1.Role)
	role.Name = tailscaleServiceAccountName
	role.Namespace = orgName
	result, err := clientutil.CreateOrPatch(ctx, k8sClient, role, func() error {
		role.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"create"},
			},
			{
				APIGroups:     []string{""},
				ResourceNames: []string{tailscaleServiceAccountName + "-auth"},
				Resources:     []string{"secrets"},
				Verbs:         []string{"get", "update", "patch"},
			},
		}
		return nil
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultNone:
		setupLog.Info("role already exists", "name", role.Name)
	case clientutil.OperationResultCreated:
		setupLog.Info("created role", "name", role.Name)
	case clientutil.OperationResultUpdated:
		setupLog.Info("updated role", "name", role.Name)
	}
	return nil
}

func createRoleBindingInRemoteCluster(ctx context.Context, k8sClient client.Client, orgName string) error {
	var roleBinding = new(rbacv1.RoleBinding)
	roleBinding.Name = tailscaleServiceAccountName
	roleBinding.Namespace = orgName
	result, err := clientutil.CreateOrPatch(ctx, k8sClient, roleBinding, func() error {
		roleBinding.Subjects = []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: tailscaleServiceAccountName,
			},
		}
		roleBinding.RoleRef = rbacv1.RoleRef{
			Kind:     "Role",
			Name:     tailscaleServiceAccountName,
			APIGroup: "rbac.authorization.k8s.io",
		}
		return nil
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultNone:
		setupLog.Info("roleBinding already exists", "name", roleBinding.Name)
	case clientutil.OperationResultCreated:
		setupLog.Info("created roleBinding", "name", roleBinding.Name)
	case clientutil.OperationResultUpdated:
		setupLog.Info("updated roleBinding", "name", roleBinding.Name)
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

func createAuthSecretInRemoteCluster(ctx context.Context, k8sClient client.Client, orgName string, preAuthKey []byte) error {
	var authSecret = new(corev1.Secret)
	authSecret.Name = "tailscale-auth"
	authSecret.Namespace = orgName

	result, err := clientutil.CreateOrPatch(ctx, k8sClient, authSecret, func() error {
		authSecret.Type = corev1.SecretTypeOpaque
		authSecret.Data = map[string][]byte{"TS_AUTHKEY": preAuthKey}
		return nil
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultNone:
		setupLog.Info("authSecret already exists", "name", authSecret.Name)
	case clientutil.OperationResultCreated:
		setupLog.Info("created authSecret", "name", authSecret.Name)
	case clientutil.OperationResultUpdated:
		setupLog.Info("updated authSecret", "name", authSecret.Name)
	}
	return nil
}

func (o *newClusterBootstrapOptions) isCalicoUsed() bool {
	dcClient, err := discovery.NewDiscoveryClientForConfig(getKubeconfigOrDie(o.customerKubeConfig))
	if err != nil {
		setupLog.Error(err, "unable to create discovery client")
		return false
	}
	apiGroupList, err := dcClient.ServerGroups()
	if err != nil {
		setupLog.Error(err, "unable to list apiGroups")
		return false
	}
	for _, apiGroup := range apiGroupList.Groups {
		if apiGroup.Name == "crd.projectcalico.org" {
			return true
		}
	}
	return false
}

func (o *newClusterBootstrapOptions) deployTailscaleInRemoteCluster(ctx context.Context, k8sClient client.Client) error {
	greenhouseTailscaleLabel := map[string]string{greenhouseapis.HeadScaleKey: "client"}
	tailscaleImageStr := o.tailscaleImage + ":" + o.tailscaleImageTag
	proxyImageStr := o.proxyImage + ":" + o.proxyImageTag
	hostNetworkBool := o.isCalicoUsed()
	setupLog.Info("hostNetworking is set to:", "hostNetwork", hostNetworkBool)
	var tailscaleDeployment = new(appsv1.Deployment)
	tailscaleDeployment.Name = o.clusterName + "tailscale"
	tailscaleDeployment.Namespace = o.orgName

	result, err := clientutil.CreateOrPatch(ctx, k8sClient, tailscaleDeployment, func() error {
		tailscaleDeployment.Labels = greenhouseTailscaleLabel
		tailscaleDeployment.Spec = appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: greenhouseTailscaleLabel},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: greenhouseTailscaleLabel,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: tailscaleServiceAccountName,
					Containers: []corev1.Container{
						{
							Name:            "proxy",
							ImagePullPolicy: corev1.PullAlways,
							Image:           proxyImageStr,
							Lifecycle: &corev1.Lifecycle{
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"/wait-shutdown",
										},
									},
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "ALL_PROXY",
									Value: "socks5://localhost:1055",
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("50m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceCPU:    resource.MustParse("5m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
							},
						},
						{
							Name:            "ts-sidecar",
							ImagePullPolicy: corev1.PullAlways,
							Image:           tailscaleImageStr,
							Args: []string{
								"--socket",
								"/tmp/tailscaled.sock",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "TS_STATE_DIR",
									Value: "/state",
								},
								{
									Name:  "TS_TAILSCALED_EXTRA_ARGS",
									Value: "--state=mem: --no-logs-no-support --debug=:8080",
								},
								{
									Name:  "TS_ACCEPT_DNS",
									Value: "false",
								},
								{
									Name:  "TS_EXTRA_ARGS",
									Value: "--login-server " + o.headscaleURI,
								},
								{
									Name:  "TS_KUBE_SECRET",
									Value: "",
								},
								{
									Name:  "TS_USERSPACE",
									Value: "true",
								},
								{
									Name:  "TS_SOCKS5_SERVER",
									Value: ":1055",
								},
								{
									Name: "TS_AUTH_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "tailscale-auth",
											},
											Key: "TS_AUTHKEY",
										},
									},
								},
							},
							LivenessProbe: &corev1.Probe{
								FailureThreshold:    5,
								SuccessThreshold:    2,
								TimeoutSeconds:      5,
								PeriodSeconds:       5,
								InitialDelaySeconds: 10,
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.FromInt32(8090),
									},
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"NET_ADMIN",
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "state",
									MountPath: "/state",
								},
							},
						},
					},
					HostNetwork: hostNetworkBool,
					Volumes: []corev1.Volume{
						{
							Name: "state",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									Medium: corev1.StorageMediumMemory,
								},
							},
						},
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultNone:
		setupLog.Info("deployment already exists", "name", tailscaleDeployment.Name)
	case clientutil.OperationResultCreated:
		setupLog.Info("created deployment", "name", tailscaleDeployment.Name)
	case clientutil.OperationResultUpdated:
		setupLog.Info("updated deployment", "name", tailscaleDeployment.Name)
	}
	return nil
}

func canI(kubeClient client.Client, namespace, user, verb, group, resourceType string) bool {
	if user == "" {
		accessReview := &authorizationv1.SelfSubjectAccessReview{
			Spec: authorizationv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Namespace: namespace,
					Verb:      verb,
					Group:     group,
					Resource:  resourceType,
				},
			},
		}

		return kubeClient.Create(ctx, accessReview) == nil && accessReview.Status.Allowed
	} else {
		accessReview := &authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Namespace: namespace,
					Verb:      verb,
					Group:     group,
					Resource:  resourceType,
				},
				User: user,
			},
		}

		return kubeClient.Create(ctx, accessReview) == nil && accessReview.Status.Allowed
	}
}

func checkPermissionMap(kubeClient client.Client, permissionMap map[string][]string, user, nameSpace string) (missingPermission []string) {
	for permissionName, permission := range permissionMap {
		if !canI(kubeClient, nameSpace, user, permission[0], permission[1], permission[2]) {
			missingPermission = append(missingPermission, permissionName)
		}
	}
	return missingPermission
}

func (o *newClusterBootstrapOptions) isClusterAlreadyBootstraped(ctx context.Context) bool {
	var cluster = new(greenhouseapisv1alpha1.Cluster)
	if err := o.ghClient.Get(ctx, client.ObjectKey{Name: o.clusterName, Namespace: o.orgName}, cluster); err != nil {
		return false
	}
	return true
}

func (o *newClusterBootstrapOptions) createClusterObject(ctx context.Context) error {
	token, err := o.createServiceAccountToken()
	if err != nil {
		return err
	}

	generateKubeconfig := &clustercontroller.KubeConfigHelper{
		Host:          o.customerConfig.Host,
		TLSServerName: o.customerConfig.TLSClientConfig.ServerName,
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
	if !o.freeAccessMode {
		clusterSecret.Labels = map[string]string{greenhouseapis.LabelAccessMode: "headscale"}
	}
	clusterSecret.Type = greenhouseapis.SecretTypeKubeConfig
	clusterSecret.Data = map[string][]byte{"kubeconfig": genKubeConfig}

	if err = o.ghClient.Create(ctx, clusterSecret); err != nil {
		return err
	}
	setupLog.Info("created clusterSecret", "name", clusterSecret.Name)

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
func (o *newClusterBootstrapOptions) isClusterObjectCreated(ctx context.Context) wait.ConditionWithContextFunc {
	return func(_ context.Context) (bool, error) {
		cluster := new(greenhouseapisv1alpha1.Cluster)
		if err := o.ghClient.Get(ctx, client.ObjectKey{Name: o.clusterName, Namespace: o.orgName}, cluster); err != nil {
			return false, err
		}

		switch cluster.Spec.AccessMode {
		case greenhouseapisv1alpha1.ClusterAccessModeDirect:
			return true, nil
		case greenhouseapisv1alpha1.ClusterAccessModeHeadscale:
			clusterSecret := new(corev1.Secret)
			if err := o.ghClient.Get(ctx, client.ObjectKey{Name: o.clusterName, Namespace: o.orgName}, clusterSecret); err != nil {
				return false, err
			}
			if clusterSecret.Data["headscalePreAuthKey"] == nil {
				return true, nil
			}
		}

		return false, nil
	}
}

func (o *newClusterBootstrapOptions) waitForClusterCreated(ctx context.Context, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, time.Second, timeout, false, o.isClusterObjectCreated(ctx))
}

func (o *newClusterBootstrapOptions) workOnCreatedCluster(ctx context.Context) error {
	if err := o.waitForClusterCreated(ctx, time.Duration(60)*time.Second); err != nil {
		return err
	}

	cluster := new(greenhouseapisv1alpha1.Cluster)
	if err := o.ghClient.Get(ctx, client.ObjectKey{Name: o.clusterName, Namespace: o.orgName}, cluster); err != nil {
		return err
	}
	if cluster.Spec.AccessMode == greenhouseapisv1alpha1.ClusterAccessModeHeadscale {
		clusterSecret := new(corev1.Secret)
		if err := o.ghClient.Get(ctx, client.ObjectKey{Name: o.clusterName, Namespace: o.orgName}, clusterSecret); err != nil {
			return err
		}
		if err := createAuthSecretInRemoteCluster(ctx, o.customerClient, o.orgName, clusterSecret.Data["headscalePreAuthKey"]); err != nil {
			return err
		}

		if err := o.deployTailscaleInRemoteCluster(ctx, o.customerClient); err != nil {
			return err
		}
	}
	return nil
}

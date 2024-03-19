// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	headscalev1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"github.com/pkg/errors"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

const tailscaleAuthorizationKey = "TS_AUTHKEY"

type HeadscaleAccessReconciler struct {
	client.Client
	recorder                               record.EventRecorder
	headscaleGRPCClient                    headscalev1.HeadscaleServiceClient
	getHeadscaleClientFromRestClientGetter func(restClientGetter genericclioptions.RESTClientGetter, proxy string, headscaleAddress string) (client.Client, error)

	HeadscaleGRPCURL,
	HeadscaleAPIKey,
	TailscaleProxy string
	// HeadscalePreAuthenticationKeyMinValidity is the minimum duration a pre-authentication has to be valid for.
	HeadscalePreAuthenticationKeyMinValidity,
	RemoteClusterBearerTokenValidity,
	RenewRemoteClusterBearerTokenAfter time.Duration
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=clusters/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *HeadscaleAccessReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)

	if r.HeadscaleGRPCURL == "" {
		return errors.New("headscale GRPC URL required but not provided")
	}
	if r.HeadscaleAPIKey == "" {
		return errors.New("headscale API key required but no provided")
	}
	if r.HeadscalePreAuthenticationKeyMinValidity == 0 {
		return errors.New("headscale pre authenticating key min validity required but no provided")
	}
	if r.RemoteClusterBearerTokenValidity == 0 {
		return errors.New("remote cluster bearer token validity required but no provided")
	}
	if r.RenewRemoteClusterBearerTokenAfter == 0 {
		return errors.New("remote cluster bearer token renewal after required but not provided")
	}
	if r.TailscaleProxy == "" {
		return errors.New("tailscale proxy required but not configured")
	}

	grpcClient, err := clientutil.NewHeadscaleGRPCClient(r.HeadscaleGRPCURL, r.HeadscaleAPIKey)
	if err != nil {
		return err
	}
	r.headscaleGRPCClient = grpcClient
	r.getHeadscaleClientFromRestClientGetter = clientutil.NewHeadscaleK8sClientFromRestClientGetter

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousev1alpha1.Cluster{}, builder.WithPredicates(
			clientutil.PredicateClusterByAccessMode(greenhousev1alpha1.ClusterAccessModeHeadscale),
		)).
		// Watch the secret owned by this cluster.
		// This should trigger a reconciliation if the user-provided or controller-generated kubeconfig was changed.
		Watches(&corev1.Secret{}, handler.EnqueueRequestForOwner(mgr.GetScheme(), mgr.GetRESTMapper(), &greenhousev1alpha1.Cluster{})).
		Complete(r)
}

/*
	The HeadscaleAccessReconciler manages clusters with access mode headscale.

	Remote clusters connected via Headscale are air-gapped from the Greenhouse central cluster,
	thus the bootstrap had to be done manually by a user with cluster-admin permissions in the remote cluster.
	The following implementation handles:
	1) User in Headscale
	2) Pre-authentication key in Headscale
	3) Namespace in remote cluster
	4) ServiceAccount in remote cluster
	5) Token for ServiceAccount in remote cluster
	5) ClusterRoleBinding in remote cluster

*/

func (r *HeadscaleAccessReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var cluster = new(greenhousev1alpha1.Cluster)
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Manage only clusters with the headscale access mode.
	// Though we use a predicate for the access mode, this check is necessary as we also watch secrets and enqueue by owner reference.
	if cluster.Spec.AccessMode != greenhousev1alpha1.ClusterAccessModeHeadscale {
		return ctrl.Result{}, nil
	}

	// Cleanup logic
	if cluster.DeletionTimestamp != nil && controllerutil.ContainsFinalizer(cluster, greenhouseapis.FinalizerCleanupCluster) {
		// Delete resources (serviceAccount, clusterRoleBinding, namespace, etc.) in remote cluster before the secret.
		isNamespaceInRemoteClusterDeleted, err := r.deleteResourcesOnRemoteCluster(ctx, cluster)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !isNamespaceInRemoteClusterDeleted {
			return ctrl.Result{RequeueAfter: defaultRequeueInterval}, nil
		}
		// Cleanup headscale resources.
		isHeadscaleMachineDeleted, err := r.deleteHeadscaleMachine(ctx, cluster)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !isHeadscaleMachineDeleted {
			return ctrl.Result{RequeueAfter: defaultRequeueInterval}, nil
		}
		isHeadscaleUserDeleted, err := r.deleteHeadscaleUser(ctx, cluster)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !isHeadscaleUserDeleted {
			return ctrl.Result{RequeueAfter: defaultRequeueInterval}, nil
		}
		// Eventually, remove the finalizer.
		err = clientutil.RemoveFinalizer(ctx, r.Client, cluster, greenhouseapis.FinalizerCleanupCluster)
		return ctrl.Result{}, err
	}

	// Ensure the cluster status is always handled.
	defer func() {
		if err := r.reconcileStatus(ctx, cluster); err != nil {
			log.FromContext(ctx).Error(err, "failed to reconcile status")
		}
	}()

	// Add finalizer before modifying any resources.
	if err := clientutil.EnsureFinalizer(ctx, r.Client, cluster, greenhouseapis.FinalizerCleanupCluster); err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile the Headscale resources for the cluster.
	if err := ReconcileHeadscaleUser(ctx, r.recorder, cluster, r.headscaleGRPCClient); err != nil {
		return ctrl.Result{}, err
	}
	// The remote cluster must have joined the headscale network before we can proceed.
	hasJoined, err := r.hasClusterJoinedHeadscale(ctx, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !hasJoined {
		r.recorder.Eventf(cluster, corev1.EventTypeNormal, "HeadscaleClusterNotJoined",
			"Waiting for the cluster to join the headscale network")
		return ctrl.Result{RequeueAfter: defaultRequeueInterval}, nil
	}

	// Reconcile the resources in the remote cluster.
	ipAddressOfHeadscaleClientInRemoteCluster, err := r.getIPAddressForHeadscaleClientInRemoteCluster(ctx, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}
	restClientGetter, err := r.newRestClientGetterForCluster(ctx, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}
	k8sHeadscaleProxyClientForRemoteCluster, err := r.getHeadscaleClientFromRestClientGetter(restClientGetter, r.TailscaleProxy, ipAddressOfHeadscaleClientInRemoteCluster)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := reconcileNamespaceInRemoteCluster(ctx, k8sHeadscaleProxyClientForRemoteCluster, cluster); err != nil {
		return ctrl.Result{}, err
	}
	if err := reconcileServiceAccountInRemoteCluster(ctx, k8sHeadscaleProxyClientForRemoteCluster, cluster); err != nil {
		return ctrl.Result{}, err
	}
	if err := reconcileClusterRoleBindingInRemoteCluster(ctx, k8sHeadscaleProxyClientForRemoteCluster, cluster); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.reconcileServiceAccountToken(ctx, restClientGetter, cluster); err != nil {
		return ctrl.Result{}, err
	}
	// The initial pre-authentication used by the remote tailscale client was provisioned during bootstrap.
	// This function only handles its timely renewal and rotation.
	if err := r.reconcilePreAuthenticationKey(ctx, k8sHeadscaleProxyClientForRemoteCluster, cluster); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: defaultRequeueInterval}, nil
}

// reconcilePreAuthenticationKey reconciles the pre-authorization key used by the tailscale client in the remote cluster.
func (r *HeadscaleAccessReconciler) reconcilePreAuthenticationKey(ctx context.Context, k8sClientForRemoteCluster client.Client, cluster *greenhousev1alpha1.Cluster) error {
	preAuthKey, err := ReconcilePreAuthorizationKey(ctx, cluster, r.headscaleGRPCClient, r.HeadscalePreAuthenticationKeyMinValidity)
	if err != nil {
		return err
	}
	var tailscaleSecretInRemoteCluster = new(corev1.Secret)
	tailscaleSecretInRemoteCluster.Namespace = cluster.GetNamespace()
	tailscaleSecretInRemoteCluster.Name = "tailscale-auth"
	result, err := clientutil.CreateOrPatch(ctx, k8sClientForRemoteCluster, tailscaleSecretInRemoteCluster, func() error {
		tailscaleSecretInRemoteCluster.Type = corev1.SecretTypeOpaque
		if tailscaleSecretInRemoteCluster.Data == nil {
			tailscaleSecretInRemoteCluster.Data = make(map[string][]byte, 0)
		}
		tailscaleSecretInRemoteCluster.Data[tailscaleAuthorizationKey] = []byte(preAuthKey.Key)
		return nil
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		r.recorder.Eventf(cluster, corev1.EventTypeNormal, "HeadscalePreAuthenticationKeyProvisioned",
			"Provisioned pre-authentication key for Headscale to secret %s", tailscaleSecretInRemoteCluster.GetName(),
		)
	case clientutil.OperationResultUpdated:
		r.recorder.Eventf(cluster, corev1.EventTypeNormal, "HeadscalePreAuthenticationKeyRenewed",
			"Renewed pre-authentication key for Headscale to secret %s", tailscaleSecretInRemoteCluster.GetName(),
		)
	}
	if err := r.updateClusterSecret(ctx, cluster, preAuthKey.Key); err != nil {
		return err
	}
	return nil
}

func (r *HeadscaleAccessReconciler) updateClusterSecret(ctx context.Context, cluster *greenhousev1alpha1.Cluster, key string) error {
	clusterSecret := new(corev1.Secret)
	clusterSecret.Namespace = cluster.GetNamespace()
	clusterSecret.Name = cluster.GetSecretName()
	result, err := clientutil.CreateOrPatch(ctx, r.Client, clusterSecret, func() error {
		clusterSecret.Type = greenhouseapis.SecretTypeKubeConfig
		if clusterSecret.Data == nil {
			clusterSecret.Data = make(map[string][]byte, 0)
		}
		clusterSecret.Data[greenhouseapis.HeadscalePreAuthKey] = []byte(key)
		return nil
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		r.recorder.Eventf(cluster, corev1.EventTypeNormal, "HeadscalePreAuthenticationKeyProvisioned",
			"Provisioned pre-authentication key for Headscale to secret %s", clusterSecret.GetName(),
		)
	case clientutil.OperationResultUpdated:
		r.recorder.Eventf(cluster, corev1.EventTypeNormal, "HeadscalePreAuthenticationKeyRenewed",
			"Renewed pre-authentication key for Headscale to secret %s", clusterSecret.GetName(),
		)
	}
	return nil
}

func (r *HeadscaleAccessReconciler) hasClusterJoinedHeadscale(ctx context.Context, cluster *greenhousev1alpha1.Cluster) (bool, error) {
	ipAddress, err := r.getIPAddressForHeadscaleClientInRemoteCluster(ctx, cluster)
	if err != nil {
		return false, err
	}
	return ipAddress != "", nil
}

func (r *HeadscaleAccessReconciler) getOnlineHeadscaleMachinesForRemoteCluster(ctx context.Context, cluster *greenhousev1alpha1.Cluster) ([]*headscalev1.Machine, error) {
	machineList, err := r.headscaleGRPCClient.ListMachines(ctx, &headscalev1.ListMachinesRequest{
		User: headscaleKeyForCluster(cluster),
	})
	if err != nil {
		return nil, err
	}
	onlineMachineList := make([]*headscalev1.Machine, 0)
	for _, machine := range machineList.GetMachines() {
		if machine.Online && len(machine.IpAddresses) > 0 && machine.User != nil && machine.User.Name == headscaleKeyForCluster(cluster) {
			onlineMachineList = append(onlineMachineList, machine)
		}
	}
	// Sort by ID highest to lowest to have the most recent machine first in the list.
	sort.SliceStable(onlineMachineList, func(i, j int) bool {
		return onlineMachineList[i].Id > onlineMachineList[j].Id
	})
	return onlineMachineList, nil
}

func (r *HeadscaleAccessReconciler) getIPAddressForHeadscaleClientInRemoteCluster(ctx context.Context, cluster *greenhousev1alpha1.Cluster) (string, error) {
	machineList, err := r.getOnlineHeadscaleMachinesForRemoteCluster(ctx, cluster)
	if err != nil {
		return "", err
	}
	for _, machine := range machineList {
		if len(machine.IpAddresses) > 0 {
			return machine.IpAddresses[0], nil
		}
	}
	return "", nil
}

func (r *HeadscaleAccessReconciler) reconcileServiceAccountToken(ctx context.Context, restClientGetter *clientutil.RestClientGetter, cluster *greenhousev1alpha1.Cluster) error {
	ipAddress, err := r.getIPAddressForHeadscaleClientInRemoteCluster(ctx, cluster)
	if err != nil {
		return err
	}
	var tokenRequestor = &tokenHelper{
		Client:                             r.Client,
		Proxy:                              r.TailscaleProxy,
		HeadscaleAddress:                   ipAddress,
		RemoteClusterBearerTokenValidity:   r.RemoteClusterBearerTokenValidity,
		RenewRemoteClusterBearerTokenAfter: r.RenewRemoteClusterBearerTokenAfter,
	}
	return tokenRequestor.ReconcileServiceAccountToken(ctx, restClientGetter, cluster)
}

// reconcileStatus updates the cluster status for transparency purposes.
// In contrast to other reconciliation functions, errors don't end the flow to ensure the status is always reported.
func (r *HeadscaleAccessReconciler) reconcileStatus(ctx context.Context, cluster *greenhousev1alpha1.Cluster) error {
	var (
		headscaleMachineStatus  = new(greenhousev1alpha1.HeadScaleMachineStatus)
		headscaleReadyCondition = greenhousev1alpha1.UnknownCondition(greenhousev1alpha1.HeadscaleReady, "", "")
	)
	if machineList, err := r.getOnlineHeadscaleMachinesForRemoteCluster(ctx, cluster); err == nil && len(machineList) > 0 {
		// TODO: I adapted the status collection from the initial implementation to have the tests green.
		// Why is only the first machine considered though?
		firstMachine := machineList[0]
		headscaleMachineStatus.ID = firstMachine.GetId()
		headscaleMachineStatus.IPAddresses = firstMachine.GetIpAddresses()
		headscaleMachineStatus.Name = firstMachine.GetName()
		headscaleMachineStatus.ForcedTags = firstMachine.GetForcedTags()
		headscaleMachineStatus.Online = firstMachine.GetOnline()
		if expirationTime := firstMachine.GetExpiry(); expirationTime != nil {
			headscaleMachineStatus.Expiry = metav1.NewTime(expirationTime.AsTime())
		}
		if createdAtTime := firstMachine.GetCreatedAt(); createdAtTime != nil {
			headscaleMachineStatus.CreatedAt = metav1.NewTime(createdAtTime.AsTime())
		}
		if preAuthKey := firstMachine.PreAuthKey; preAuthKey != nil {
			headscaleMachineStatus.PreAuthKey = &greenhousev1alpha1.PreAuthKey{
				ID:        preAuthKey.GetId(),
				User:      preAuthKey.GetUser(),
				Reusable:  preAuthKey.GetReusable(),
				Ephemeral: preAuthKey.GetEphemeral(),
				Used:      preAuthKey.GetUsed(),
			}
			if createdAtTime := preAuthKey.GetCreatedAt(); createdAtTime != nil {
				headscaleMachineStatus.PreAuthKey.CreatedAt = metav1.NewTime(createdAtTime.AsTime())
			}
			if expirationTime := preAuthKey.GetExpiration(); expirationTime != nil {
				headscaleMachineStatus.PreAuthKey.Expiration = metav1.NewTime(expirationTime.AsTime())
			}
		}
		headscaleReadyCondition.Status = metav1.ConditionTrue
	} else if len(machineList) == 0 {
		headscaleReadyCondition.Status = metav1.ConditionFalse
		headscaleReadyCondition.Message = "no headscale machine found"
		log.FromContext(ctx).Error(nil, "no headscale machine found", "cluster", cluster.GetName(), "namespace", cluster.GetNamespace())
	} else {
		headscaleReadyCondition.Status = metav1.ConditionFalse
		headscaleReadyCondition.Message = err.Error()
		log.FromContext(ctx).Error(err, "failed to get headscale machine status")
	}

	var kubernetesVersion = "unknown"
	if restClientGetterForRemoteCluster, err := r.newRestClientGetterForCluster(ctx, cluster); err == nil {
		// Get the kubernetes version of the remote cluster.
		if k8sVersion, err := clientutil.GetKubernetesVersion(restClientGetterForRemoteCluster); err == nil {
			kubernetesVersion = k8sVersion.String()
		} else {
			log.FromContext(ctx).Error(err, "failed to get cluster version")
		}
	} else {
		log.FromContext(ctx).Error(err, "failed to get rest client getter for cluster")
	}

	_, err := clientutil.PatchStatus(ctx, r.Client, cluster, func() error {
		cluster.Status.KubernetesVersion = kubernetesVersion
		cluster.Status.HeadScaleStatus = headscaleMachineStatus
		cluster.Status.SetConditions(headscaleReadyCondition)
		return nil
	})
	return err
}

// deleteHeadscaleMachine deletes all headscale machines for the given cluster and returns true if none can be found.
// Note: This function is meant to be re-tried on false or an error.
func (r *HeadscaleAccessReconciler) deleteHeadscaleMachine(ctx context.Context, cluster *greenhousev1alpha1.Cluster) (bool, error) {
	machinesToDelete, err := r.headscaleGRPCClient.ListMachines(ctx, &headscalev1.ListMachinesRequest{
		User: headscaleKeyForCluster(cluster),
	})
	if err != nil {
		return false, err
	}
	// Report done if there's no machine associated with the cluster.
	if len(machinesToDelete.GetMachines()) == 0 {
		return true, nil
	}
	allErrs := make([]error, 0)
	for _, machine := range machinesToDelete.GetMachines() {
		if _, err := r.headscaleGRPCClient.DeleteMachine(ctx, &headscalev1.DeleteMachineRequest{
			MachineId: machine.GetId(),
		}); err != nil {
			allErrs = append(allErrs, err)
		}
	}
	// We return false here to indicate this function should be called again as deleting machines in Headscale might take a while.
	return false, utilerrors.NewAggregate(allErrs)
}

func (r *HeadscaleAccessReconciler) deleteHeadscaleUser(ctx context.Context, cluster *greenhousev1alpha1.Cluster) (bool, error) {
	// Check if the user still exists.
	_, err := r.headscaleGRPCClient.GetUser(ctx, &headscalev1.GetUserRequest{
		Name: headscaleKeyForCluster(cluster),
	})
	if err != nil {
		errStatus, ok := status.FromError(err)
		if !ok {
			return false, err
		}
		if strings.Contains(errStatus.Message(), "not found") {
			return true, nil
		}
		return false, err
	}
	// Attempt deletion of the user.
	if _, err := r.headscaleGRPCClient.DeleteUser(ctx, &headscalev1.DeleteUserRequest{
		Name: headscaleKeyForCluster(cluster),
	}); err != nil {
		errStatus, ok := status.FromError(err)
		if !ok {
			return false, err
		}
		if strings.Contains(errStatus.Message(), "not found") {
			return true, nil
		}
		return false, err
	}
	return true, nil
}

func (r *HeadscaleAccessReconciler) deleteResourcesOnRemoteCluster(ctx context.Context, cluster *greenhousev1alpha1.Cluster) (bool, error) {
	ipAddress, err := r.getIPAddressForHeadscaleClientInRemoteCluster(ctx, cluster)
	if err != nil {
		return false, err
	}
	restClientGetter, err := r.newRestClientGetterForCluster(ctx, cluster)
	if err != nil {
		return false, err
	}
	k8sHeadscaleProxyClientForRemoteCluster, err := r.getHeadscaleClientFromRestClientGetter(restClientGetter, r.TailscaleProxy, ipAddress)
	if err != nil {
		return false, err
	}
	// all remote resources are bound by owner-reference to the namespace
	if err := deleteNamespaceInRemoteCluster(ctx, k8sHeadscaleProxyClientForRemoteCluster, cluster); err != nil {
		return false, err
	}
	return true, nil
}

func (r *HeadscaleAccessReconciler) newRestClientGetterForCluster(ctx context.Context, cluster *greenhousev1alpha1.Cluster) (*clientutil.RestClientGetter, error) {
	var clusterSecret = new(corev1.Secret)
	if err := r.Get(ctx, types.NamespacedName{Name: cluster.GetSecretName(), Namespace: cluster.GetNamespace()}, clusterSecret); err != nil {
		return nil, err
	}
	return clientutil.NewRestClientGetterFromSecret(clusterSecret, cluster.Namespace)
}

// headscaleKeyForCluster returns the key for the given cluster to use in headscale.
func headscaleKeyForCluster(cluster *greenhousev1alpha1.Cluster) string {
	return fmt.Sprintf("%s-%s", cluster.GetNamespace(), cluster.GetName())
}

// isPreAuthenticationKeyIsNotExpired returns true if the given pre-authentication key is valid and not yet expired.
func isPreAuthenticationKeyIsNotExpired(preAuthenticationKey *headscalev1.PreAuthKey, preAuthenticationKeyMinValidity time.Duration) bool {
	if preAuthenticationKey.Expiration == nil {
		return false
	}
	return preAuthenticationKey.Expiration.IsValid() &&
		preAuthenticationKey.Expiration.AsTime().After(time.Now().Add(preAuthenticationKeyMinValidity))
}

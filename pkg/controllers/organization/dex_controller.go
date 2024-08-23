// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package organization

import (
	"context"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/dexidp/dex/connector/oidc"
	"github.com/pkg/errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greenhousesapv1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/common"
	dexapi "github.com/cloudoperators/greenhouse/pkg/dex/api"
	"github.com/cloudoperators/greenhouse/pkg/util"
)

const dexConnectorTypeGreenhouse = "greenhouse-oidc"

// DexReconciler reconciles a Organization object
type DexReconciler struct {
	client.Client
	recorder  record.EventRecorder
	Namespace string
}

//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=greenhouse.sap,resources=organizations/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=dex.coreos.com,resources=connectors;oauth2clients,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// SetupWithManager sets up the controller with the Manager.
func (r *DexReconciler) SetupWithManager(name string, mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor(name)
	if r.Namespace == "" {
		return errors.New("namespace required but missing")
	}
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&greenhousesapv1alpha1.Organization{}).
		Owns(&dexapi.Connector{}).
		Owns(&dexapi.OAuth2Client{}).
		// Watch secrets referenced by organizations for confidential values.
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.enqueueOrganizationForReferencedSecret)).
		Complete(r)
}

func (r *DexReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var org = new(greenhousesapv1alpha1.Organization)
	if err := r.Get(ctx, req.NamespacedName, org); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Ignore organizations without OIDC configuration.
	if org.Spec.Authentication == nil || org.Spec.Authentication.OIDCConfig == nil {
		return ctrl.Result{}, nil
	}

	if err := r.reconcileDexConnector(ctx, org); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileOAuth2Client(ctx, org); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *DexReconciler) reconcileDexConnector(ctx context.Context, org *greenhousesapv1alpha1.Organization) error {
	clientID, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, r.Client, org.Name, org.Spec.Authentication.OIDCConfig.ClientIDReference)
	if err != nil {
		return err
	}
	clientSecret, err := clientutil.GetSecretKeyFromSecretKeyReference(ctx, r.Client, org.Name, org.Spec.Authentication.OIDCConfig.ClientSecretReference)
	if err != nil {
		return err
	}
	redirectURL, err := r.discoverOIDCRedirectURL(ctx, org)
	if err != nil {
		return err
	}
	oidcConfig := &oidc.Config{
		Issuer:               org.Spec.Authentication.OIDCConfig.Issuer,
		ClientID:             clientID,
		ClientSecret:         clientSecret,
		RedirectURI:          redirectURL,
		UserNameKey:          "login_name",
		UserIDKey:            "login_name",
		InsecureSkipVerify:   true,
		InsecureEnableGroups: true,
	}
	configByte, err := json.Marshal(oidcConfig)
	if err != nil {
		return err
	}
	var dexConnector = new(dexapi.Connector)
	dexConnector.Namespace = r.Namespace
	dexConnector.ObjectMeta.Name = org.Name
	result, err := clientutil.CreateOrPatch(ctx, r.Client, dexConnector, func() error {
		dexConnector.DexConnector.Type = dexConnectorTypeGreenhouse
		dexConnector.DexConnector.Name = cases.Title(language.English).String(org.Name)
		dexConnector.DexConnector.ID = org.Name
		dexConnector.DexConnector.Config = configByte
		return controllerutil.SetControllerReference(org, dexConnector, r.Scheme())
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created dex connector", "namespace", dexConnector.Namespace, "name", dexConnector.GetName())
		r.recorder.Eventf(org, corev1.EventTypeNormal, "CreatedDexConnector", "Created dex connector %s/%s", dexConnector.Namespace, dexConnector.GetName())
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated dex connector", "namespace", dexConnector.Namespace, "name", dexConnector.GetName())
		r.recorder.Eventf(org, corev1.EventTypeNormal, "UpdatedDexConnector", "Updated dex connector %s/%s", dexConnector.Namespace, dexConnector.GetName())
	}
	return nil
}

func (r *DexReconciler) enqueueOrganizationForReferencedSecret(_ context.Context, o client.Object) []ctrl.Request {
	var org = new(greenhousesapv1alpha1.Organization)
	if err := r.Get(context.Background(), types.NamespacedName{Namespace: "", Name: o.GetNamespace()}, org); err != nil {
		return nil
	}
	return []ctrl.Request{{NamespacedName: client.ObjectKeyFromObject(org)}}
}

func (r *DexReconciler) discoverOIDCRedirectURL(ctx context.Context, org *greenhousesapv1alpha1.Organization) (string, error) {
	if r := org.Spec.Authentication.OIDCConfig.RedirectURI; r != "" {
		return r, nil
	}
	var ingressList = new(networkingv1.IngressList)
	if err := r.List(ctx, ingressList, client.InNamespace(r.Namespace), client.MatchingLabels{"app.kubernetes.io/name": "idproxy"}); err != nil {
		return "", err
	}
	for _, ing := range ingressList.Items {
		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				return ensureCallbackURL(rule.Host), nil
			}
		}
	}
	return "", errors.New("oidc redirect URL not provided and cannot be discovered")
}

func (r *DexReconciler) reconcileOAuth2Client(ctx context.Context, org *greenhousesapv1alpha1.Organization) error {
	var oAuth2Client = new(dexapi.OAuth2Client)
	oAuth2Client.ObjectMeta.Name = encodedOAuth2ClientName(org.Name)
	oAuth2Client.ObjectMeta.Namespace = r.Namespace

	result, err := clientutil.CreateOrPatch(ctx, r.Client, oAuth2Client, func() error {
		oAuth2Client.Client.Public = true
		oAuth2Client.Client.ID = org.Name
		oAuth2Client.Client.Name = org.Name
		if oAuth2Client.RedirectURIs == nil {
			oAuth2Client.RedirectURIs = make([]string, 2)
		}
		// Ensure the required redirect URLs are present.
		// Additional ones can be added by the user.
		for _, requiredRedirectURL := range []string{
			"https://dashboard." + common.DNSDomain,
			fmt.Sprintf("https://%s.dashboard.%s", org.Name, common.DNSDomain),
		} {
			oAuth2Client.Client.RedirectURIs = util.AppendStringToSliceIfNotContains(requiredRedirectURL, oAuth2Client.RedirectURIs)
		}
		return controllerutil.SetControllerReference(org, oAuth2Client, r.Scheme())
	})
	if err != nil {
		return err
	}
	switch result {
	case clientutil.OperationResultCreated:
		log.FromContext(ctx).Info("created oauth2client", "namespace", oAuth2Client.Namespace, "name", oAuth2Client.GetName())
		r.recorder.Eventf(org, corev1.EventTypeNormal, "CreatedOAuth2Client", "Created oauth2client %s/%s", oAuth2Client.Namespace, oAuth2Client.GetName())
	case clientutil.OperationResultUpdated:
		log.FromContext(ctx).Info("updated oauth2client", "namespace", oAuth2Client.Namespace, "name", oAuth2Client.GetName())
		r.recorder.Eventf(org, corev1.EventTypeNormal, "UpdatedOAuth2Client", "Updated oauth2client %s/%s", oAuth2Client.Namespace, oAuth2Client.GetName())
	}
	return nil
}

func ensureCallbackURL(url string) string {
	prefix := "https://"
	if !strings.HasPrefix(url, prefix) {
		url = prefix + url
	}
	suffix := "/callback"
	if !strings.HasSuffix(url, suffix) {
		url += suffix
	}
	return url
}

func encodedOAuth2ClientName(orgName string) string {
	// See https://github.com/dexidp/dex/issues/1606 for encoding.
	return strings.TrimRight(base32.
		NewEncoding("abcdefghijklmnopqrstuvwxyz234567").
		EncodeToString(fnv.New64().Sum([]byte(orgName))), "=",
	)
}

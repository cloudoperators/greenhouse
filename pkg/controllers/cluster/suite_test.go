// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package cluster_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	v1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/cloudoperators/greenhouse/pkg/admission"
	clusterpkg "github.com/cloudoperators/greenhouse/pkg/controllers/cluster"
	"github.com/cloudoperators/greenhouse/pkg/test"
)

var (
	remoteCfg        *rest.Config
	remoteClient     client.Client
	remoteKubeConfig []byte
	remoteEnvTest    *envtest.Environment

	tailscaleProxyURL = "https://127.0.0.1:8080"

	headscaleReconciler *clusterpkg.HeadscaleAccessReconciler
	bootstrapReconciler *clusterpkg.BootstrapReconciler
)

func TestClusterBootstrap(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ClusterControllerSuite")
}

var _ = BeforeSuite(func() {

	bootstrapReconciler = &clusterpkg.BootstrapReconciler{}
	test.RegisterController("clusterBootstrap", (bootstrapReconciler).SetupWithManager)
	test.RegisterController("clusterDirectAccess", (&clusterpkg.DirectAccessReconciler{
		RemoteClusterBearerTokenValidity:   10 * time.Minute,
		RenewRemoteClusterBearerTokenAfter: 9 * time.Minute,
	}).SetupWithManager)
	headscaleReconciler = &clusterpkg.HeadscaleAccessReconciler{
		HeadscaleGRPCURL:                         "willBeMocked",
		HeadscaleAPIKey:                          "willBeMocked",
		TailscaleProxy:                           tailscaleProxyURL,
		HeadscalePreAuthenticationKeyMinValidity: 5 * time.Minute,
		RemoteClusterBearerTokenValidity:         10 * time.Minute,
		RenewRemoteClusterBearerTokenAfter:       9 * time.Minute,
	}
	test.RegisterController("clusterHeadscaleAccess", (headscaleReconciler).SetupWithManager)
	test.RegisterController("clusterStatus", (&clusterpkg.ClusterStatusReconciler{}).SetupWithManager)
	test.RegisterWebhook("clusterValidation", admission.SetupClusterWebhookWithManager)
	test.RegisterWebhook("secretsWebhook", admission.SetupSecretWebhookWithManager)

	test.TestBeforeSuite()

	// inject mocks
	fakeHeadscaleGRPCClient := newFakeHeadscaleClient()
	clusterpkg.ExportSetHeadscaleGRPCClientOnHAR(headscaleReconciler, fakeHeadscaleGRPCClient)
	getterFunc := newFakeHeadscaleClientGetter
	clusterpkg.ExportSetRestClientGetterFunc(headscaleReconciler, getterFunc)

	By("bootstrapping remote cluster")
	bootstrapRemoteCluster()

	/*
		This is commented as the access to the remote cluster requires a https proxy.
			Though the proxy is in-place, golang does not account for a proxy on localhost (1) and
		injecting custom transport in the client.Client is not supported when using TLS certificates (2).
		(1) https://maelvls.dev/go-ignores-proxy-localhost,
		(2) https://github.com/kubernetes/client-go/blob/master/transport/transport.go#L38-L40.

		go func() {
			if err := runReverseProxy(test.Ctx, tailscaleProxyURL, headscaleEnvTest); err != nil {
				log.Fatalf("Server error: %v", err)
			}
		}()
	*/
})

var _ = AfterSuite(func() {
	By("tearing down the test environment and remote cluster")
	test.TestAfterSuite()
	Expect(remoteEnvTest.Stop()).
		NotTo(HaveOccurred(), "there must be no error stopping the remote environment")
})

func bootstrapRemoteCluster() {
	remoteCfg, remoteClient, remoteEnvTest, remoteKubeConfig = test.StartControlPlane("6885", false, false)
}

func newFakeHeadscaleClientGetter(_ genericclioptions.RESTClientGetter, _, _ string) (client.Client, error) {
	/*
		This is commented as the access to the remote cluster requires a https proxy.
			Though the proxy is in-place, golang does not account for a proxy on localhost (1) and
		injecting custom transport in the client.Client is not supported when using TLS certificates (2).
		(1) https://maelvls.dev/go-ignores-proxy-localhost,
		(2) https://github.com/kubernetes/client-go/blob/master/transport/transport.go#L38-L40.

		cfgTransportCfg, err := headscaleCfg.TransportConfig()
		if err != nil {
			return nil, err
		}
		tlsCfg, err := transport.TLSConfigFor(cfgTransportCfg)
		if err != nil {
			return nil, err
		}
		headscaleCfg.Transport = &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse(tailscaleProxyURL)
			},
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			TLSClientConfig:       tlsCfg,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}*/
	return remoteClient, nil
}

// newFakeHeadscaleClient mocks the Headscale GRPC client and returns the configured responses
func newFakeHeadscaleClient() v1.HeadscaleServiceClient {
	cl := test.FakeHeadscaleClient{
		IsUserDeleted: false,
	}
	cl.GetUserFunc = func(ctx context.Context, in *v1.GetUserRequest, opts ...grpc.CallOption) (*v1.GetUserResponse, error) {
		if cl.IsUserDeleted {
			return nil, status.Errorf(2, "User not found")
		}
		return &v1.GetUserResponse{User: &v1.User{
			Id:        "1",
			Name:      in.GetName(),
			CreatedAt: timestamppb.Now(),
		}}, nil
	}
	cl.CreateUserFunc = func(ctx context.Context, request *v1.CreateUserRequest, option ...grpc.CallOption) (*v1.CreateUserResponse, error) {
		return &v1.CreateUserResponse{User: &v1.User{Id: "1", Name: request.GetName(), CreatedAt: timestamppb.Now()}}, nil
	}
	cl.ListMachinesFunc = func(ctx context.Context, in *v1.ListMachinesRequest, opts ...grpc.CallOption) (*v1.ListMachinesResponse, error) {
		return &v1.ListMachinesResponse{Machines: []*v1.Machine{{
			Id:                   uint64(1337),
			MachineKey:           "machine",
			NodeKey:              "node",
			DiscoKey:             "disco",
			IpAddresses:          []string{"127.0.0.1"},
			Name:                 "myMachine",
			User:                 &v1.User{Id: "1", Name: in.GetUser(), CreatedAt: timestamppb.Now()},
			LastSeen:             timestamppb.Now(),
			LastSuccessfulUpdate: timestamppb.Now(),
			Expiry:               timestamppb.New(time.Now().Add(10 * time.Minute)),
			PreAuthKey:           newFakePreAuthKey(in.GetUser()),
			CreatedAt:            timestamppb.Now(),
			RegisterMethod:       v1.RegisterMethod_REGISTER_METHOD_AUTH_KEY,
			ForcedTags:           []string{"tag:one"},
			InvalidTags:          nil,
			ValidTags:            nil,
			GivenName:            "myMachine",
			Online:               true,
		}}}, nil
	}
	cl.CreatePreAuthKeyFunc = func(ctx context.Context, in *v1.CreatePreAuthKeyRequest, opts ...grpc.CallOption) (*v1.CreatePreAuthKeyResponse, error) {
		return &v1.CreatePreAuthKeyResponse{
			PreAuthKey: newFakePreAuthKey(in.GetUser()),
		}, nil
	}
	cl.ListPreAuthKeysFunc = func(ctx context.Context, request *v1.ListPreAuthKeysRequest, option ...grpc.CallOption) (*v1.ListPreAuthKeysResponse, error) {
		return &v1.ListPreAuthKeysResponse{
			PreAuthKeys: []*v1.PreAuthKey{newFakePreAuthKey(request.GetUser())},
		}, nil
	}
	cl.DeleteMachineFunc = func(ctx context.Context, in *v1.DeleteMachineRequest, opts ...grpc.CallOption) (*v1.DeleteMachineResponse, error) {
		return &v1.DeleteMachineResponse{}, nil
	}
	cl.DeleteUserFunc = func(ctx context.Context, in *v1.DeleteUserRequest, opts ...grpc.CallOption) (*v1.DeleteUserResponse, error) {
		cl.IsUserDeleted = true
		return &v1.DeleteUserResponse{}, nil
	}
	return cl
}

func newFakePreAuthKey(userName string) *v1.PreAuthKey {
	return &v1.PreAuthKey{
		User:       userName,
		Id:         "1",
		Key:        "someKey",
		Reusable:   false,
		Ephemeral:  false,
		Used:       false,
		Expiration: timestamppb.New(time.Now().Add(10 * time.Minute)),
		CreatedAt:  timestamppb.Now(),
		AclTags:    nil,
	}
}

//nolint:unused // See comments on unit test with proxy on localhost using TLS certificates.
func runReverseProxy(ctx context.Context, proxyAddress string, testEnv *envtest.Environment) error {
	testEnvAPIServerConfig := testEnv.ControlPlane.APIServer
	if testEnvAPIServerConfig == nil {
		return errors.New("the test environment has no api server configured")
	}
	remote, err := url.Parse(fmt.Sprintf("http://%s", net.JoinHostPort(testEnvAPIServerConfig.SecureServing.Address, testEnvAPIServerConfig.SecureServing.Port)))
	if err != nil {
		return err
	}
	handler := func(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			r.Host = remote.Host
			p.ServeHTTP(w, r)
		}
	}
	tlsX509KeyPair, err := tls.LoadX509KeyPair(
		filepath.Join(testEnvAPIServerConfig.CertDir, "apiserver.crt"),
		filepath.Join(testEnvAPIServerConfig.CertDir, "apiserver.key"),
	)
	if err != nil {
		return err
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(testEnvAPIServerConfig.CA) {
		return fmt.Errorf("failed to append CA certs")
	}
	//nolint:gosec // I promise to not use that in production.
	tlsConfig := &tls.Config{
		RootCAs:      certPool,
		Certificates: []tls.Certificate{tlsX509KeyPair},
	}
	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Transport = &http.Transport{TLSClientConfig: tlsConfig}
	//nolint:forbidigo // I promise to not use that in production.
	http.HandleFunc("/", handler(proxy))
	server := &http.Server{TLSConfig: tlsConfig}
	proxyURL, err := url.Parse(proxyAddress)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp4", proxyURL.Host)
	if err != nil {
		return err
	}
	defer listener.Close()
	tlsListener := tls.NewListener(listener, tlsConfig)
	defer tlsListener.Close()

	go func() {
		//nolint:gosimple // I promise to not use that in production.
		select {
		case <-ctx.Done():
			if err := server.Shutdown(ctx); err != nil {
				fmt.Printf("error shutting reverse proxy: %v\n", err)
			}
		}
	}()
	if err := server.Serve(tlsListener); err != nil {
		fmt.Println(err.Error())
	}
	<-ctx.Done()
	return nil
}

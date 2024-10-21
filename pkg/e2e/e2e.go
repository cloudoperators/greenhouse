package e2e

import (
	"context"
	"fmt"
	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/internal/local/klient"
	"github.com/cloudoperators/greenhouse/pkg/internal/local/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AdminKubeConfigPathEnv     = "GREENHOUSE_ADMIN_KUBECONFIG"
	RemoteExtKubeConfigPathEnv = "REMOTE_EXT_KUBECONFIG"
	RemoteIntKubeConfigPathEnv = "REMOTE_INT_KUBECONFIG"
)

type clientBox struct {
	client     client.Client
	restClient *clientutil.RestClientGetter
}

type TestEnv struct {
	adminClusterClient       *clientBox
	remoteExtClusterClient   *clientBox
	TestNamespace            string
	RemoteIntKubeConfigBytes []byte
	TestOrganization         *greenhousev1alpha1.Organization
}

func NewExecutionEnv(userScheme ...func(s *runtime.Scheme) error) *TestEnv {
	adminClusterClient, err := prepareClients(AdminKubeConfigPathEnv, userScheme...)
	utils.CheckError(err)
	remoteExtClusterClient, err := prepareClients(RemoteExtKubeConfigPathEnv)
	utils.CheckError(err)
	remoteIntKubeCfgPath, err := fromEnv(RemoteIntKubeConfigPathEnv)
	utils.CheckError(err)
	remoteIntKubeCfgBytes, err := readFileContent(remoteIntKubeCfgPath)
	utils.CheckError(err)
	return &TestEnv{
		adminClusterClient:       adminClusterClient,
		remoteExtClusterClient:   remoteExtClusterClient,
		RemoteIntKubeConfigBytes: remoteIntKubeCfgBytes,
	}
}

func (env *TestEnv) WithOrganization(ctx context.Context, samplePath string) *TestEnv {
	org := &greenhousev1alpha1.Organization{}
	orgBytes, err := readFileContent(samplePath)
	utils.CheckError(err)
	err = utils.FromYamlToK8sObject(string(orgBytes), org)
	utils.CheckError(err)
	err = env.adminClusterClient.client.Create(ctx, org)
	utils.CheckError(err)
	err = utils.WaitUntilNamespaceCreated(ctx, env.adminClusterClient.client, org.Name)
	utils.CheckError(err)
	env.TestNamespace = org.Name
	env.TestOrganization = org
	return env
}

func (env *TestEnv) WithRemoteClusterOnboarding(ctx context.Context) *TestEnv {
	remoteIntKubeCfgPath, err := fromEnv(RemoteIntKubeConfigPathEnv)
	utils.CheckError(err)
	remoteIntKubeCfgBytes, err := readFileContent(remoteIntKubeCfgPath)
	utils.CheckError(err)
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "remote-int-cluster",
			Namespace: env.TestNamespace,
		},
		Type: greenhouseapis.SecretTypeKubeConfig,
		Data: map[string][]byte{
			greenhouseapis.KubeConfigKey: remoteIntKubeCfgBytes,
		},
	}
	err = env.adminClusterClient.client.Create(ctx, secret)
	utils.CheckError(err)
	return env
}

func (env *TestEnv) GetAdminClient() client.Client {
	return env.adminClusterClient.client
}

func (env *TestEnv) GetAdminRESTClient() *clientutil.RestClientGetter {
	return env.adminClusterClient.restClient
}

func prepareClients(kubeconfigEnv string, userScheme ...func(s *runtime.Scheme) error) (*clientBox, error) {
	kubeconfigPath, err := fromEnv(kubeconfigEnv)
	if err != nil {
		return nil, err
	}
	kubeconfigBytes, err := readFileContent(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	restConfig, k8sClient, err := klient.NewKubeClientFromConfigWithScheme(string(kubeconfigBytes), userScheme...)
	if err != nil {
		panic(fmt.Sprintf("error creating admin CRUD client: %v", err))
	}
	return &clientBox{
		client:     k8sClient,
		restClient: clientutil.NewRestClientGetterFromRestConfig(restConfig, ""),
	}, nil
}

func readFileContent(path string) ([]byte, error) {
	_, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func fromEnv(key string) (string, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("environment variable %s not set", key)
	}
	return val, nil
}

func FromYaml(path string, resource ...any) error {
	return utils.FromYamlToK8sObject(path, resource...)
}

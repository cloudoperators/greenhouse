package e2e

import (
	"bytes"
	"context"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	greenhouseapis "github.com/cloudoperators/greenhouse/pkg/apis"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/internal/local/klient"
	"github.com/cloudoperators/greenhouse/pkg/internal/local/utils"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
	"io"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

const (
	AdminKubeConfigPathEnv     = "GREENHOUSE_ADMIN_KUBECONFIG"
	RemoteKubeConfigPathEnv    = "GREENHOUSE_REMOTE_KUBECONFIG"
	remoteIntKubeConfigPathEnv = "GREENHOUSE_REMOTE_INT_KUBECONFIG"
	ControllerLogsPathEnv      = "CONTROLLER_LOGS_PATH"
	managerDeploymentName      = "greenhouse-controller-manager"
	managerDeploymentNamespace = "greenhouse"
	remoteExecutionEnv         = "EXECUTION_ENV"
	realCluster                = "GARDENER"
)

const (
	AdminClient      EClient = "AdminClient"
	RemoteClient     EClient = "RemoteClient"
	AdminRESTClient  EClient = "AdminRESTClient"
	RemoteRESTClient EClient = "RemoteRESTClient"
	AdminClientSet   EClient = "AdminClientSet"
	RemoteClientSet  EClient = "RemoteClientSet"
)

var defaultElapsedTime = 30 * time.Second

type EClient string
type WaitApplyFunc func(resource lifecycle.RuntimeObject) error

type clientBox struct {
	client     client.Client
	restClient *clientutil.RestClientGetter
	clientSet  kubernetes.Interface
}

type TestEnv struct {
	adminClusterClient    *clientBox
	remoteClusterClient   *clientBox
	TestNamespace         string
	IsRealCluster         bool
	RemoteKubeConfigBytes []byte
}

func NewExecutionEnv(userScheme ...func(s *runtime.Scheme) error) *TestEnv {
	adminClusterClient, err := prepareClients(AdminKubeConfigPathEnv, userScheme...)
	utils.CheckError(err)
	remoteClusterClient, err := prepareClients(RemoteKubeConfigPathEnv)
	utils.CheckError(err)
	isReal := isRealCluster()
	var remoteKubeCfgBytes []byte
	var remoteKubeCfgPath string
	if isReal {
		remoteKubeCfgPath, err = fromEnv(RemoteKubeConfigPathEnv)
		utils.CheckError(err)
		remoteKubeCfgBytes, err = readFileContent(remoteKubeCfgPath)
		utils.CheckError(err)
	} else {
		remoteIntKubeCfgPath, err := fromEnv(remoteIntKubeConfigPathEnv)
		utils.CheckError(err)
		remoteKubeCfgBytes, err = readFileContent(remoteIntKubeCfgPath)
		utils.CheckError(err)
	}
	return &TestEnv{
		adminClusterClient:    adminClusterClient,
		remoteClusterClient:   remoteClusterClient,
		RemoteKubeConfigBytes: remoteKubeCfgBytes,
		IsRealCluster:         isReal,
	}
}

func isRealCluster() bool {
	execEnv, ok := os.LookupEnv(remoteExecutionEnv)
	if !ok {
		return false
	}
	return strings.TrimSpace(execEnv) == realCluster
}

func (env *TestEnv) WithOrganization(ctx context.Context, samplePath string) *TestEnv {
	org := &greenhousev1alpha1.Organization{}
	orgBytes, err := readFileContent(samplePath)
	utils.CheckError(err)
	err = utils.FromYamlToK8sObject(string(orgBytes), org)
	utils.CheckError(err)
	err = env.adminClusterClient.client.Create(ctx, org)
	if client.IgnoreAlreadyExists(err) != nil {
		utils.CheckError(err)
	}
	err = utils.WaitUntilNamespaceCreated(ctx, env.adminClusterClient.client, org.Name)
	utils.CheckError(err)
	env.TestNamespace = org.Name
	return env
}

func (env *TestEnv) WithOnboardedRemoteCluster(ctx context.Context, name string) *TestEnv {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: env.TestNamespace,
		},
		Type: greenhouseapis.SecretTypeKubeConfig,
		Data: map[string][]byte{
			greenhouseapis.KubeConfigKey: env.RemoteKubeConfigBytes,
		},
	}
	err := env.adminClusterClient.client.Create(ctx, secret)
	utils.CheckError(err)
	return env
}

func (env *TestEnv) GetClient(clientType EClient) client.Client {
	switch clientType {
	case AdminClient:
		return env.adminClusterClient.client
	case RemoteClient:
		return env.remoteClusterClient.client
	default:
		utils.CheckError(fmt.Errorf("client type %s not supported", clientType))
		return nil
	}
}

func (env *TestEnv) GetRESTClient(clientType EClient) *clientutil.RestClientGetter {
	switch clientType {
	case AdminRESTClient:
		return env.adminClusterClient.restClient
	case RemoteRESTClient:
		return env.remoteClusterClient.restClient
	default:
		utils.CheckError(fmt.Errorf("client type %s not supported", clientType))
		return nil
	}
}

func (env *TestEnv) GetClientSet(clientType EClient) kubernetes.Interface {
	switch clientType {
	case AdminClientSet:
		return env.adminClusterClient.clientSet
	case RemoteClientSet:
		return env.remoteClusterClient.clientSet
	default:
		utils.CheckError(fmt.Errorf("client type %s not supported", clientType))
		return nil
	}
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
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return &clientBox{
		client:     k8sClient,
		restClient: clientutil.NewRestClientGetterFromRestConfig(restConfig, ""),
		clientSet:  clientSet,
	}, nil
}

func GetRESTConfigFromBytes(kubeconfigBytes []byte) (*rest.Config, error) {
	return clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
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

func WaitUntilResourceReadyOrNotReady(ctx context.Context, apiClient client.Client, resource lifecycle.RuntimeObject, name, namespace string, applyFunc WaitApplyFunc, readyStatus bool) error {
	b := backoff.NewExponentialBackOff(backoff.WithInitialInterval(5*time.Second), backoff.WithMaxElapsedTime(defaultElapsedTime))
	return backoff.Retry(func() error {
		utils.Log("waiting for resource to be ready... \n")
		err := apiClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, resource)
		if err != nil {
			return err
		}
		if applyFunc != nil {
			err = applyFunc(resource)
			if err != nil {
				return err
			}
		}
		conditions := resource.GetConditions()
		readyCondition := conditions.GetConditionByType(greenhousev1alpha1.ReadyCondition).IsTrue()
		expected := readyCondition == readyStatus
		utils.Logf("readyCondition: %v, expectedStatus: %v, calcualted: %v\n", readyCondition, readyStatus, expected)
		if !expected {
			return fmt.Errorf("resource %s is not yet in expected state", resource.GetName())
		}
		return nil
	}, b)
}

func (env *TestEnv) GenerateControllerLogs(ctx context.Context, startTime time.Time) {
	podLogsPath, err := fromEnv(ControllerLogsPathEnv)
	if err != nil {
		utils.Logf("%s", err.Error())
		return
	}

	k8sClient := env.adminClusterClient.client
	clientSet := env.adminClusterClient.clientSet
	deployment := &appsv1.Deployment{}

	err = k8sClient.Get(ctx, client.ObjectKey{Name: managerDeploymentName, Namespace: managerDeploymentNamespace}, deployment)
	if err != nil {
		utils.Logf("error getting deployment: %s", err.Error())
		return
	}

	pods := &corev1.PodList{}
	err = k8sClient.List(ctx, pods, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(deployment.Spec.Selector.MatchLabels),
		Namespace:     deployment.Namespace,
	})
	if err != nil {
		utils.Logf("error listing pods: %s", err.Error())
		return
	}
	if len(pods.Items) < 0 {
		utils.Logf("no pods found for deployment %s", managerDeploymentName)
		return
	}

	podName := pods.Items[0].Name
	podLogOpts := corev1.PodLogOptions{
		Container: "manager",
		SinceTime: &metav1.Time{Time: startTime},
	}
	req := clientSet.CoreV1().Pods(managerDeploymentNamespace).GetLogs(podName, &podLogOpts)
	logStream, err := req.Stream(ctx)
	if err != nil {
		utils.Logf("error getting pod logs stream %s", err.Error())
		return
	}
	defer func(logStream io.ReadCloser) {
		err := logStream.Close()
		if err != nil {
			utils.Logf("error closing pod logs stream in defer %s", err.Error())
		}
	}(logStream)

	// Create or open the log file
	file, err := os.Create(podLogsPath)
	if err != nil {
		utils.Logf("error creating log file %s: %s", podLogsPath, err.Error())
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			utils.Logf("error closing log file in defer %s", err.Error())
		}
	}(file)

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, logStream)
	if err != nil {
		utils.Logf("error copying pod logs %s", err.Error())
		return
	}
	// Write the logs to the file
	_, err = file.WriteString(buf.String())
	if err != nil {
		utils.Logf("error writing pod logs to file %s: %s", podLogsPath, err.Error())
		return
	}
	utils.Logf("pod %s logs written to file: %s", podName, podLogsPath)
}

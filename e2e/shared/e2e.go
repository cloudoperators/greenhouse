// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package shared

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
	"github.com/cloudoperators/greenhouse/pkg/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
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

type EClient string

const (
	AdminClient      EClient = "AdminClient"
	RemoteClient     EClient = "RemoteClient"
	AdminRESTClient  EClient = "AdminRESTClient"
	RemoteRESTClient EClient = "RemoteRESTClient"
	AdminClientSet   EClient = "AdminClientSet"
	RemoteClientSet  EClient = "RemoteClientSet"
)

var defaultElapsedTime = 180 * time.Second

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
	Expect(err).NotTo(HaveOccurred(), "error preparing admin cluster client")
	remoteClusterClient, err := prepareClients(RemoteKubeConfigPathEnv)
	Expect(err).NotTo(HaveOccurred(), "error preparing remote cluster client")

	isReal := isRealCluster()
	var remoteKubeCfgBytes []byte
	var remoteKubeCfgPath string
	if isReal {
		GinkgoWriter.Printf("Running on real cluster\n")
		remoteKubeCfgPath, err = fromEnv(RemoteKubeConfigPathEnv)
		Expect(err).NotTo(HaveOccurred(), "error getting remote kubeconfig path")
		remoteKubeCfgBytes, err = readFileContent(remoteKubeCfgPath)
		Expect(err).NotTo(HaveOccurred(), "error reading remote kubeconfig file")
	} else {
		GinkgoWriter.Printf("Running on local cluster\n")
		remoteIntKubeCfgPath, err := fromEnv(remoteIntKubeConfigPathEnv)
		Expect(err).NotTo(HaveOccurred(), "error getting remote internal kubeconfig path")
		remoteKubeCfgBytes, err = readFileContent(remoteIntKubeCfgPath)
		Expect(err).NotTo(HaveOccurred(), "error reading remote internal kubeconfig file")
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
	Expect(err).NotTo(HaveOccurred(), "error reading organization sample data")
	err = FromYamlToK8sObject(string(orgBytes), org)
	Expect(err).NotTo(HaveOccurred(), "error converting organization yaml to k8s object")

	err = env.adminClusterClient.client.Create(ctx, org)
	Expect(client.IgnoreAlreadyExists(err)).NotTo(HaveOccurred(), "error creating organization")

	// TODO: check ready condition on organization after standardization
	err = WaitUntilNamespaceCreated(ctx, env.adminClusterClient.client, org.Name)
	Expect(err).NotTo(HaveOccurred(), "error waiting for namespace to be created")
	env.TestNamespace = org.Name
	return env
}

func (env *TestEnv) GetClient(clientType EClient) client.Client {
	switch clientType {
	case AdminClient:
		return env.adminClusterClient.client
	case RemoteClient:
		return env.remoteClusterClient.client
	default:
		GinkgoWriter.Printf("client type %s not supported", clientType)
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
		GinkgoWriter.Printf("client type %s not supported", clientType)
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
		GinkgoWriter.Printf("client type %s not supported", clientType)
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
	restConfig, k8sClient, err := NewKubeClientFromConfigWithScheme(string(kubeconfigBytes), userScheme...)
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

func IsResourceOwnedByOwner(owner, owned metav1.Object) bool {
	runtimeObj, ok := (owner).(runtime.Object)
	if !ok {
		return false
	}
	// ClusterRoleBinding does not have a type information (So We add it)
	if runtimeObj.GetObjectKind().GroupVersionKind() == (schema.GroupVersionKind{}) {
		if err := addTypeInformationToObject(runtimeObj); err != nil {
			GinkgoWriter.Printf("error adding type information to object: %s", err.Error())
			return false
		}
	}
	for _, ownerRef := range owned.GetOwnerReferences() {
		if ownerRef.Name == owner.GetName() && ownerRef.UID == owner.GetUID() && ownerRef.Kind == runtimeObj.GetObjectKind().GroupVersionKind().Kind {
			return true
		}
	}
	return false
}

func addTypeInformationToObject(obj runtime.Object) error {
	gvks, _, err := scheme.Scheme.ObjectKinds(obj)
	if err != nil {
		return fmt.Errorf("missing apiVersion or kind and cannot assign it; %w", err)
	}

	for _, gvk := range gvks {
		if gvk.Kind == "" || gvk.Version == "" || gvk.Version == runtime.APIVersionInternal {
			continue
		}
		obj.GetObjectKind().SetGroupVersionKind(gvk)
		break
	}
	return nil
}

func WaitUntilResourceReadyOrNotReady(ctx context.Context, apiClient client.Client, resource lifecycle.RuntimeObject, name, namespace string, applyFunc WaitApplyFunc, readyStatus bool) error {
	b := backoff.NewExponentialBackOff(backoff.WithInitialInterval(5*time.Second), backoff.WithMaxElapsedTime(defaultElapsedTime))
	return backoff.Retry(func() error {
		Logf("waiting for resource %s to be ready... \n", name)
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
		readyCondition := conditions.GetConditionByType(greenhousev1alpha1.ReadyCondition)
		if readyCondition == nil {
			return fmt.Errorf("resource %s does not have ready condition yet", resource.GetName())
		}
		expected := readyCondition.IsTrue() == readyStatus
		Logf("readyCondition: %v, expectedStatus: %v, calculated: %v\n", readyCondition.IsTrue(), readyStatus, expected)
		if !expected {
			return fmt.Errorf("resource %s is not yet in expected state", resource.GetName())
		}
		return nil
	}, b)
}

func WaitUntilNamespaceCreated(ctx context.Context, k8sClient client.Client, name string) error {
	b := backoff.NewExponentialBackOff(backoff.WithInitialInterval(5*time.Second), backoff.WithMaxElapsedTime(30*time.Second))
	return backoff.Retry(func() error {
		Logf("waiting for namespace %s to be created...", name)
		ns := &corev1.Namespace{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name: name,
		}, ns)
		if err != nil {
			return err
		}
		if ns.Status.Phase != corev1.NamespaceActive {
			return errors.New("namespace is not yet ready")
		}
		return nil
	}, b)
}

func (env *TestEnv) GenerateControllerLogs(ctx context.Context, startTime time.Time) {
	podLogsPath, err := fromEnv(ControllerLogsPathEnv)
	if err != nil {
		Logf("%s", err.Error())
		return
	}

	k8sClient := env.adminClusterClient.client
	clientSet := env.adminClusterClient.clientSet
	deployment := &appsv1.Deployment{}

	err = k8sClient.Get(ctx, client.ObjectKey{Name: managerDeploymentName, Namespace: managerDeploymentNamespace}, deployment)
	if err != nil {
		Logf("error getting deployment: %s", err.Error())
		return
	}

	pods := &corev1.PodList{}
	err = k8sClient.List(ctx, pods, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(deployment.Spec.Selector.MatchLabels),
		Namespace:     deployment.Namespace,
	})
	if err != nil {
		Logf("error listing pods: %s", err.Error())
		return
	}
	if len(pods.Items) == 0 {
		Logf("no pods found for deployment %s", managerDeploymentName)
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
		Logf("error getting pod logs stream %s", err.Error())
		return
	}
	defer func(logStream io.ReadCloser) {
		err := logStream.Close()
		if err != nil {
			Logf("error closing pod logs stream in defer %s", err.Error())
		}
	}(logStream)

	// Create or open the log file
	file, err := os.Create(podLogsPath)
	if err != nil {
		Logf("error creating log file %s: %s", podLogsPath, err.Error())
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			Logf("error closing log file in defer %s", err.Error())
		}
	}(file)

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, logStream)
	if err != nil {
		Logf("error copying pod logs %s", err.Error())
		return
	}
	// Write the logs to the file
	_, err = file.WriteString(buf.String())
	if err != nil {
		Logf("error writing pod logs to file %s: %s", podLogsPath, err.Error())
		return
	}
	Logf("pod %s logs written to file: %s", podName, podLogsPath)
}

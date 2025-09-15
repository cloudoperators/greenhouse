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
	"path/filepath"
	"strings"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/cenkalti/backoff/v5"
	"k8s.io/apimachinery/pkg/types"

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

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/internal/lifecycle"
)

const (
	AdminKubeConfigPathEnv     = "GREENHOUSE_ADMIN_KUBECONFIG"
	RemoteKubeConfigPathEnv    = "GREENHOUSE_REMOTE_KUBECONFIG"
	remoteIntKubeConfigPathEnv = "GREENHOUSE_REMOTE_INT_KUBECONFIG"
	ControllerLogsPathEnv      = "CONTROLLER_LOGS_PATH"
	managerDeploymentName      = "greenhouse-controller-manager"
	managerDeploymentNamespace = "greenhouse"
	fluxDeploymentNamespace    = "flux-system"
	fluxDeploymentName         = "helm-controller"
	remoteExecutionEnv         = "EXECUTION_ENV"
	realCluster                = "GARDENER"

	// Define retry timeout for when backoff should stop
	maxRetries = 10
)

type WaitApplyFunc func(resource lifecycle.RuntimeObject) error

type TestEnv struct {
	AdminRestClientGetter  *clientutil.RestClientGetter
	RemoteRestClientGetter *clientutil.RestClientGetter
	TestNamespace          string
	IsRealCluster          bool
	RemoteKubeConfigBytes  []byte
}

func NewExecutionEnv() *TestEnv {
	adminGetter := clientGetter(AdminKubeConfigPathEnv)
	remoteGetter := clientGetter(RemoteKubeConfigPathEnv)

	isReal := isRealCluster()
	var err error
	var remoteKubeCfgBytes []byte
	var remoteKubeCfgPath string
	if isReal {
		Log("Running on real cluster\n")
		remoteKubeCfgPath, err = fromEnv(RemoteKubeConfigPathEnv)
		Expect(err).NotTo(HaveOccurred(), "error getting remote kubeconfig path")
		remoteKubeCfgBytes, err = readFileContent(remoteKubeCfgPath)
		Expect(err).NotTo(HaveOccurred(), "error reading remote kubeconfig file")
	} else {
		Log("Running on local cluster\n")
		remoteIntKubeCfgPath, err := fromEnv(remoteIntKubeConfigPathEnv)
		Expect(err).NotTo(HaveOccurred(), "error getting remote internal kubeconfig path")
		remoteKubeCfgBytes, err = readFileContent(remoteIntKubeCfgPath)
		Expect(err).NotTo(HaveOccurred(), "error reading remote internal kubeconfig file")
	}
	return &TestEnv{
		AdminRestClientGetter:  adminGetter,
		RemoteRestClientGetter: remoteGetter,
		RemoteKubeConfigBytes:  remoteKubeCfgBytes,
		IsRealCluster:          isReal,
	}
}

func isRealCluster() bool {
	execEnv, ok := os.LookupEnv(remoteExecutionEnv)
	if !ok {
		return false
	}
	return strings.TrimSpace(execEnv) == realCluster
}

func (env *TestEnv) WithOrganization(ctx context.Context, k8sClient client.Client, samplePath string) *TestEnv {
	org := &greenhousev1alpha1.Organization{}
	orgBytes, err := readFileContent(samplePath)
	Expect(err).NotTo(HaveOccurred(), "error reading organization sample data")
	err = FromYamlToK8sObject(string(orgBytes), org)
	Expect(err).NotTo(HaveOccurred(), "error converting organization yaml to k8s object")
	Logf("creating organization %s", org.Name)
	err = k8sClient.Create(ctx, org)
	Expect(client.IgnoreAlreadyExists(err)).NotTo(HaveOccurred(), "error creating organization")

	// TODO: check ready condition on organization after standardization
	err = WaitUntilNamespaceCreated(ctx, k8sClient, org.Name)
	Expect(err).NotTo(HaveOccurred(), "error waiting for namespace to be created")
	env.TestNamespace = org.Name
	return env
}

func clientGetter(kubeconfigEnv string) *clientutil.RestClientGetter {
	kubeconfigPath, err := fromEnv(kubeconfigEnv)
	Expect(err).NotTo(HaveOccurred(), "error getting kubeconfig path from env")
	kubeconfigBytes, err := readFileContent(kubeconfigPath)
	Expect(err).NotTo(HaveOccurred(), "error reading kubeconfig file")
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	Expect(err).NotTo(HaveOccurred(), "error getting rest config from kubeconfig")
	return clientutil.NewRestClientGetterFromRestConfig(config, "")
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
			LogErr("error adding type information to object: %s", err.Error())
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

func getStandardBackoff() *backoff.ExponentialBackOff {
	// Create an exponential backoff instance
	b := &backoff.ExponentialBackOff{
		InitialInterval:     500 * time.Millisecond, // Start with 500ms delay
		RandomizationFactor: 0.5,                    // Randomize interval by ±50%
		Multiplier:          2.0,                    // Double the interval each time
		MaxInterval:         15 * time.Second,       // Cap at 15s between retries
	}
	return b
}

func WaitUntilResourceReadyOrNotReady(ctx context.Context, apiClient client.Client, resource lifecycle.RuntimeObject, name, namespace string, applyFunc WaitApplyFunc, readyStatus bool) error {
	// Create an exponential backoff instance
	b := getStandardBackoff()
	b.Reset() // Ensure backoff starts fresh
	// Track retry count
	retries := 0

	// Define the operation function
	operation := func() (op bool, err error) {
		if retries >= maxRetries {
			err = backoff.Permanent(fmt.Errorf("resource %s did not become ready after %d retries", name, maxRetries))
			return
		}

		Logf("waiting for resource %s to be ready... (attempt %d)\n", name, retries+1)

		err = apiClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, resource)
		if err != nil {
			retries++
			return
		}

		if applyFunc != nil {
			err = applyFunc(resource)
			if err != nil {
				retries++
				return
			}
		}

		conditions := resource.GetConditions()
		readyCondition := conditions.GetConditionByType(greenhousemetav1alpha1.ReadyCondition)
		if readyCondition == nil {
			retries++
			err = fmt.Errorf("resource %s does not have ready condition yet", resource.GetName())
			return
		}

		expected := readyCondition.IsTrue() == readyStatus
		Logf("readyCondition: %v, expectedStatus: %v, calculated: %v\n", readyCondition.IsTrue(), readyStatus, expected)

		if !expected {
			retries++
			err = fmt.Errorf("resource %s is not yet in expected state", resource.GetName())
		}
		op = true // success
		return
	}

	// Run the operation with backoff retry
	_, err := backoff.Retry(ctx, operation, backoff.WithBackOff(b))
	return err
}

// WaitUntilNamespaceCreated waits until the namespace is created and active
// TODO: Remove this once organization controller is standardized
func WaitUntilNamespaceCreated(ctx context.Context, k8sClient client.Client, name string) error {
	b := getStandardBackoff()
	b.Reset() // Ensure backoff starts fresh
	retries := 0
	op := func() (op bool, err error) {
		if retries >= maxRetries {
			err = backoff.Permanent(fmt.Errorf("namespace %s did not become ready after %d retries", name, maxRetries))
			return
		}
		Logf("waiting for namespace %s to be created...", name)
		ns := &corev1.Namespace{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name: name,
		}, ns)
		if err != nil {
			retries++
			return
		}
		if ns.Status.Phase != corev1.NamespaceActive {
			retries++
			err = errors.New("namespace is not yet ready")
			return
		}
		op = true
		return
	}
	_, err := backoff.Retry(ctx, op, backoff.WithBackOff(b))
	return err
}

func (env *TestEnv) GenerateControllerLogs(ctx context.Context, startTime time.Time) {
	podLogsPath, err := fromEnv(ControllerLogsPathEnv)
	if err != nil {
		Logf("%s", err.Error())
		return
	}

	config, err := env.AdminRestClientGetter.ToRESTConfig()
	if err != nil {
		Logf("error getting admin rest config: %s", err.Error())
		return
	}
	k8sClient, err := clientutil.NewK8sClientFromRestClientGetter(env.AdminRestClientGetter)
	if err != nil {
		Logf("error creating k8s client: %s", err.Error())
		return
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		Logf("error creating k8s clientset: %s", err.Error())
		return
	}
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

func (env *TestEnv) GenerateFluxHelmControllerLogs(ctx context.Context, startTime time.Time) {
	path, err := fromEnv(ControllerLogsPathEnv)
	if err != nil {
		Logf("%s", err.Error())
		return
	}

	podLogsPath := filepath.Join(filepath.Dir(path), "flux-"+filepath.Base(path))

	config, err := env.AdminRestClientGetter.ToRESTConfig()
	if err != nil {
		Logf("error getting admin rest config: %s", err.Error())
		return
	}
	k8sClient, err := clientutil.NewK8sClientFromRestClientGetter(env.AdminRestClientGetter)
	if err != nil {
		Logf("error creating k8s client: %s", err.Error())
		return
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		Logf("error creating k8s clientset: %s", err.Error())
		return
	}
	deployment := &appsv1.Deployment{}

	err = k8sClient.Get(ctx, client.ObjectKey{Name: fluxDeploymentName, Namespace: fluxDeploymentNamespace}, deployment)
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
		Logf("no pods found for deployment %s", fluxDeploymentName)
		return
	}

	podName := pods.Items[0].Name
	podLogOpts := corev1.PodLogOptions{
		Container: "manager",
		SinceTime: &metav1.Time{Time: startTime},
	}
	req := clientSet.CoreV1().Pods(fluxDeploymentNamespace).GetLogs(podName, &podLogOpts)
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

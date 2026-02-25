// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package shared

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/cenkalti/backoff/v5"
	"k8s.io/apimachinery/pkg/types"

	"github.com/google/go-github/v82/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	greenhousemetav1alpha1 "github.com/cloudoperators/greenhouse/api/meta/v1alpha1"
	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
	"github.com/cloudoperators/greenhouse/internal/clientutil"
	"github.com/cloudoperators/greenhouse/pkg/lifecycle"
)

const (
	AdminKubeConfigPathEnv     = "GREENHOUSE_ADMIN_KUBECONFIG"
	RemoteKubeConfigPathEnv    = "GREENHOUSE_REMOTE_KUBECONFIG"
	remoteIntKubeConfigPathEnv = "GREENHOUSE_REMOTE_INT_KUBECONFIG"
	ControllerLogsPathEnv      = "CONTROLLER_LOGS_PATH"
	managerDeploymentName      = "greenhouse-controller-manager"
	managerDeploymentNamespace = "greenhouse"
	managerContainer           = "manager"
	fluxDeploymentNamespace    = "flux-system"
	remoteExecutionEnv         = "EXECUTION_ENV"
	executionRuntimeEnv        = "RUNTIME_ENV"
	runtimeCI                  = "CI"
	realCluster                = "GARDENER"

	// Define retry timeout for when backoff should stop
	maxRetries = 10
)

const (
	envGitHubToken          = "GH_TOKEN"
	envGitHubAppID          = "GH_APP_ID"
	envGitHubInstallationID = "GH_APP_INSTALLATION_ID"
	envGitHubAppPrivateKey  = "GH_APP_PRIVATE_KEY"
)

type SecretType int

const (
	GitHubSecretTypePAT SecretType = iota
	GitHubSecretTypeAPP
	GitHubSecretTypeFake
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
		remoteKubeCfgBytes, err = os.ReadFile(remoteKubeCfgPath)
		Expect(err).NotTo(HaveOccurred(), "error reading remote kubeconfig file")
	} else {
		Log("Running on local cluster\n")
		remoteIntKubeCfgPath, err := fromEnv(remoteIntKubeConfigPathEnv)
		Expect(err).NotTo(HaveOccurred(), "error getting remote internal kubeconfig path")
		remoteKubeCfgBytes, err = os.ReadFile(remoteIntKubeCfgPath)
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
	orgBytes, err := os.ReadFile(samplePath)
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
	kubeconfigBytes, err := os.ReadFile(kubeconfigPath)
	Expect(err).NotTo(HaveOccurred(), "error reading kubeconfig file")
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	Expect(err).NotTo(HaveOccurred(), "error getting rest config from kubeconfig")
	return clientutil.NewRestClientGetterFromRestConfig(config, "")
}

func fromEnv(key string) (string, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("environment variable %s not set", key)
	}
	return val, nil
}

func IsResourceOwnedByOwner(owner, owned metav1.Object) bool {
	runtimeObj, ok := (owner).(kruntime.Object)
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

func addTypeInformationToObject(obj kruntime.Object) error {
	gvks, _, err := scheme.Scheme.ObjectKinds(obj)
	if err != nil {
		return fmt.Errorf("missing apiVersion or kind and cannot assign it; %w", err)
	}

	for _, gvk := range gvks {
		if gvk.Kind == "" || gvk.Version == "" || gvk.Version == kruntime.APIVersionInternal {
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

func (env *TestEnv) GenerateGreenhouseControllerLogs(ctx context.Context, startTime time.Time) {
	path, err := fromEnv(ControllerLogsPathEnv)
	if err != nil {
		Logf("%s", err.Error())
		return
	}
	scenario, err := fromEnv("SCENARIO")
	if err != nil {
		Logf("%s", err.Error())
		return
	}
	baseDir := filepath.Dir(path)
	podLogsPath := filepath.Join(baseDir, "greenhouse-"+scenario+"-e2e-pod-logs.txt")
	writeDeploymentLogs(ctx, env.AdminRestClientGetter,
		managerDeploymentNamespace, managerDeploymentName,
		managerContainer, podLogsPath, startTime)
}

func (env *TestEnv) GenerateFluxControllerLogs(ctx context.Context, controllerName string, startTime time.Time) {
	path, err := fromEnv(ControllerLogsPathEnv)
	if err != nil {
		Logf("error reading %s: %s", ControllerLogsPathEnv, err.Error())
		return
	}

	// Extract scenario from env if available
	scenario, err := fromEnv("SCENARIO")
	if err != nil {
		Logf("error reading SCENARIO env: %s", err.Error())
		return
	}

	// Use the directory of CONTROLLER_LOGS_PATH to build our Flux log filename
	baseDir := filepath.Dir(path)
	podLogsPath := filepath.Join(baseDir, fmt.Sprintf("flux-%s-e2e-%s.txt", scenario, controllerName))

	writeDeploymentLogs(ctx, env.AdminRestClientGetter,
		fluxDeploymentNamespace, controllerName,
		managerContainer, podLogsPath, startTime)
}

func writeDeploymentLogs(
	ctx context.Context,
	getter *clientutil.RestClientGetter,
	deployNS, deployName, container, outPath string,
	start time.Time,
) {

	if outPath == "" {
		Logf("no output path provided, skipping logs for %s/%s", deployNS, deployName)
		return
	}

	config, err := getter.ToRESTConfig()
	if err != nil {
		Logf("error getting rest config: %s", err.Error())
		return
	}
	k8sClient, err := clientutil.NewK8sClientFromRestClientGetter(getter)
	if err != nil {
		Logf("error creating k8s client: %s", err.Error())
		return
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		Logf("error creating k8s clientset: %s", err.Error())
		return
	}

	deploy := &appsv1.Deployment{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: deployName, Namespace: deployNS}, deploy); err != nil {
		Logf("error getting deployment %s/%s: %s", deployNS, deployName, err.Error())
		return
	}

	pods := &corev1.PodList{}
	if err := k8sClient.List(ctx, pods, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(deploy.Spec.Selector.MatchLabels),
		Namespace:     deployNS,
	}); err != nil {
		Logf("error listing pods for %s/%s: %s", deployNS, deployName, err.Error())
		return
	}
	if len(pods.Items) == 0 {
		Logf("no pods found for deployment %s/%s", deployNS, deployName)
		return
	}

	podName := pods.Items[0].Name
	opts := &corev1.PodLogOptions{
		Container: container,
		SinceTime: &metav1.Time{Time: start},
	}
	req := clientSet.CoreV1().Pods(deployNS).GetLogs(podName, opts)

	stream, err := req.Stream(ctx)
	if err != nil {
		Logf("error opening log stream for pod %s/%s (container=%s): %s", deployNS, podName, container, err.Error())
		return
	}
	defer func() {
		if e := stream.Close(); e != nil {
			Logf("error closing log stream: %s", e.Error())
		}
	}()

	// Create (truncate) file and write all logs
	file, err := os.Create(outPath)
	if err != nil {
		Logf("error creating log file %s: %s", outPath, err.Error())
		return
	}
	defer func() {
		if e := file.Close(); e != nil {
			Logf("error closing log file: %s", e.Error())
		}
	}()

	if _, err := io.Copy(file, stream); err != nil {
		Logf("error copying logs to file %s: %s", outPath, err.Error())
		return
	}
	Logf("pod %s logs written to file: %s", podName, outPath)
}

func (env *TestEnv) WithGitHubSecret(ctx context.Context, k8sClient client.Client, name string, secretType SecretType) *TestEnv {
	switch secretType {
	case GitHubSecretTypeFake:
		secret := &corev1.Secret{}
		secret.SetName(name)
		secret.SetNamespace(env.TestNamespace)
		secret.Data = map[string][]byte{
			"username": []byte("fake-user"),
			"password": []byte("fake-token"),
		}
		err := k8sClient.Create(ctx, secret)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				err = k8sClient.Get(ctx, client.ObjectKeyFromObject(secret), secret)
				Expect(err).NotTo(HaveOccurred(), "error getting existing fake github secret - "+name)
				secret.Data = map[string][]byte{
					"username": []byte("fake-user"),
					"password": []byte("fake-token"),
				}
				err = k8sClient.Update(ctx, secret)
				Expect(err).NotTo(HaveOccurred(), "error updating existing fake github secret - "+name)
				return env
			}
		}
		Expect(err).NotTo(HaveOccurred(), "error creating fake github secret - "+name)
		return env
	case GitHubSecretTypePAT:
		return env.withGithubTokenSecret(ctx, k8sClient, name)
	case GitHubSecretTypeAPP:
		ci, ok := os.LookupEnv(executionRuntimeEnv)
		if ok && strings.TrimSpace(ci) == runtimeCI {
			return env.withGitHubAppSecret(ctx, k8sClient, name)
		}
		GinkgoWriter.Printf("GitHub App secrets are only set in CI environment, falling back to PAT secret")
		return env.withGithubTokenSecret(ctx, k8sClient, name)
	default:
		log.Fatalf("unsupported secret type: %v", secretType)
	}
	return nil
}

func (env *TestEnv) withGitHubAppSecret(ctx context.Context, k8sClient client.Client, name string) *TestEnv {
	appID, appOk := os.LookupEnv(envGitHubAppID)
	installationID, inOk := os.LookupEnv(envGitHubInstallationID)
	privateKey, keyOk := os.LookupEnv(envGitHubAppPrivateKey)
	if !appOk || !inOk || !keyOk {
		log.Fatalf("one of env %s, %s or %s is not found. Set them and re-run test", envGitHubAppID, envGitHubInstallationID, envGitHubAppPrivateKey)
	}
	if strings.TrimSpace(appID) == "" || strings.TrimSpace(installationID) == "" || strings.TrimSpace(privateKey) == "" {
		log.Fatalf("one of env %s, %s or %s is empty. Set them and re-run test", envGitHubAppID, envGitHubInstallationID, envGitHubAppPrivateKey)
	}
	secret := &corev1.Secret{}
	secret.SetName(name)
	secret.SetNamespace(env.TestNamespace)
	secret.Data = map[string][]byte{
		"githubAppID":             []byte(appID),
		"githubAppInstallationID": []byte(installationID),
		"githubAppPrivateKey":     []byte(privateKey),
	}
	err := k8sClient.Create(ctx, secret)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			err = k8sClient.Get(ctx, client.ObjectKeyFromObject(secret), secret)
			Expect(err).NotTo(HaveOccurred(), "error getting existing github secret - "+name)
			secret.Data = map[string][]byte{
				"githubAppID":             []byte(appID),
				"githubAppInstallationID": []byte(installationID),
				"githubAppPrivateKey":     []byte(privateKey),
			}
			err = k8sClient.Update(ctx, secret)
			Expect(err).NotTo(HaveOccurred(), "error updating existing github secret - "+name)
			return env
		}
	}
	Expect(err).NotTo(HaveOccurred(), "error creating github secret - "+name)
	return env
}

func (env *TestEnv) withGithubTokenSecret(ctx context.Context, k8sClient client.Client, name string) *TestEnv {
	var err error
	token := os.Getenv(envGitHubToken)
	if token == "" {
		log.Fatal("env GH_TOKEN not found. 'export GH_TOKEN=<your-github-token>' and re-run test")
	}
	username := getGitHubTokenUserName(ctx, token)
	secret := &corev1.Secret{}
	secret.SetName(name)
	secret.SetNamespace(env.TestNamespace)
	secret.Data = map[string][]byte{
		"username": []byte(username),
		"password": []byte(token),
	}
	err = k8sClient.Create(ctx, secret)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			err = k8sClient.Get(ctx, client.ObjectKeyFromObject(secret), secret)
			Expect(err).NotTo(HaveOccurred(), "error getting existing github secret - "+name)
			secret.Data = map[string][]byte{
				"username": []byte(username),
				"password": []byte(token),
			}
			err = k8sClient.Update(ctx, secret)
			Expect(err).NotTo(HaveOccurred(), "error updating existing github secret - "+name)
			return env
		}
	}
	Expect(err).NotTo(HaveOccurred(), "error creating github secret - "+name)
	return env
}

func getGitHubTokenUserName(ctx context.Context, token string) string {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	ghClient := github.NewClient(tc)
	user, _, err := ghClient.Users.Get(ctx, "")
	Expect(err).NotTo(HaveOccurred(), "error getting github token user")
	return user.GetLogin()
}

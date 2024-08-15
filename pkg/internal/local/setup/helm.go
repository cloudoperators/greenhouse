package setup

import (
	"context"
	"errors"
	"github.com/cloudoperators/greenhouse/pkg/internal/local/utils"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
)

type HelmOptions struct {
	ClusterName    *string
	CurrentContext bool
	ReleaseName    string
	Namespace      string
	ChartPath      string
	ValuesPath     *string
	KubeConfigPath *string
}

type helmValues map[string]interface{}

type HelmClientOption func(*HelmOptions)

// WithKubeConfigPath - sets the kubeConfigPath flag for the helm client
// kubeconfig path will be provided to helm action configuration
// Note: Absolute paths are preferred
func WithKubeConfigPath(kubeConfigPath string) HelmClientOption {
	return func(h *HelmOptions) {
		h.KubeConfigPath = utils.StringP(kubeConfigPath)
	}
}

// WithChartPath - sets the chartPath flag for the helm client
// Note: Absolute paths are preferred
func WithChartPath(chartPath string) HelmClientOption {
	return func(h *HelmOptions) {
		h.ChartPath = chartPath
	}
}

// WithClusterName - sets the clusterName flag for the helm client
// used in a kind cluster scenario. By providing a kind cluster name,
// the kubeconfig will be fetched for the kind cluster using kind.GetKubeCfg(clusterName, false)
func WithClusterName(clusterName string) HelmClientOption {
	return func(h *HelmOptions) {
		h.ClusterName = utils.StringP(clusterName)
	}
}

// WithReleaseName - sets the releaseName flag for the helm client
// release name will be used to install the chart or render the template with release labels
func WithReleaseName(releaseName string) HelmClientOption {
	return func(h *HelmOptions) {
		h.ReleaseName = releaseName
	}
}

// WithNamespace - sets the namespace flag for the helm client
// namespace will be used to install the chart or render the template
func WithNamespace(namespace string) HelmClientOption {
	return func(h *HelmOptions) {
		h.Namespace = namespace
	}
}

// WithValuesPath - sets the valuesPath flag for the helm client
// values provided in the file will be used to render the chart
// if no values path is provided, the default values will be used from util.GetManagerHelmValues()
func WithValuesPath(valuesPath string) HelmClientOption {
	return func(h *HelmOptions) {
		h.ValuesPath = utils.StringP(valuesPath)
	}
}

// WithCurrentContext - sets the currentContext flag for the helm client
// no kubeconfig path will be provided in this case to helm action configuration,
// the current context set by the user will be used
func WithCurrentContext(currentContext bool) HelmClientOption {
	return func(h *HelmOptions) {
		h.CurrentContext = currentContext
	}
}

// apply - applies the HelmOptions to the helmClient
func apply(options *HelmOptions) *helmClient {
	return &helmClient{
		clusterName:    options.ClusterName,
		currentContext: options.CurrentContext,
		chartPath:      options.ChartPath,
		releaseName:    options.ReleaseName,
		namespace:      options.Namespace,
		valuesPath:     options.ValuesPath,
		kubeconfigPath: options.KubeConfigPath,
	}
}

type helmClient struct {
	install           *action.Install
	upgrade           *action.Upgrade
	currentContext    bool
	clusterName       *string
	releaseName       string
	namespace         string
	chartPath         string
	values            map[string]interface{}
	valuesPath        *string
	kubeconfigPath    *string
	tmpKubeConfigPath *string
}

// IHelm - interface wrapper for an actual Helm client
type IHelm interface {
	Install(ctx context.Context) (string, error)
	Template(ctx context.Context) (string, error)
	GetKubeconfigPath() *string
	GetReleaseNamespace() string
	GetClusterName() string
}

// NewHelmClient - creates a new Helm client with given HelmClientOption options
// currently supporting helm install and template actions and can be extended to support other actions
func NewHelmClient(ctx context.Context, opts ...HelmClientOption) (IHelm, error) {
	logger := utils.NewKLog(ctx)
	options := &HelmOptions{}
	for _, opt := range opts {
		opt(options)
	}
	hc := apply(options)

	if hc.kubeconfigPath == nil && hc.clusterName == nil && !hc.currentContext {
		return nil, errors.New("either kubeconfig or cluster name must be provided")
	}

	if hc.releaseName == "" {
		return nil, errors.New("release name must be provided")
	}

	if hc.namespace == "" {
		return nil, errors.New("namespace must be provided")
	}

	if hc.chartPath == "" {
		return nil, errors.New("chart path must be provided")
	}

	if hc.valuesPath == nil {
		hc.values = utils.GetManagerHelmValues()
	}

	flags := &genericclioptions.ConfigFlags{}

	if !hc.currentContext {
		if hc.kubeconfigPath == nil {
			configStr, err := GetKubeCfg(*hc.clusterName, false)
			if err != nil {
				return nil, err
			}
			tmpKubeConfigPath, err := utils.WriteToTmpFolder(*hc.clusterName, configStr)
			if err != nil {
				return nil, err
			}
			hc.kubeconfigPath = &tmpKubeConfigPath
			hc.tmpKubeConfigPath = &tmpKubeConfigPath
		}
		flags.KubeConfig = hc.kubeconfigPath
	}
	flags.Namespace = &hc.namespace

	actionConfig := new(action.Configuration)
	err := actionConfig.Init(
		flags,
		hc.namespace,
		"secret",
		logger.V(10).Info,
	)
	if err != nil {
		return nil, err
	}
	hc.install = action.NewInstall(actionConfig)
	hc.upgrade = action.NewUpgrade(actionConfig)
	return hc, nil
}

// Install - installs the helm chart
func (h *helmClient) Install(ctx context.Context) (string, error) {
	c, vals, err := h.prepareChartAndValues()
	if err != nil {
		return "", err
	}
	rel, err := h.install.RunWithContext(ctx, c, vals)
	if err != nil {
		return "", err
	}
	return rel.Manifest, nil
}

// Template - returns the rendered template of the chart
func (h *helmClient) Template(ctx context.Context) (string, error) {
	h.install.ReleaseName = h.releaseName
	h.install.Namespace = h.namespace
	h.install.DryRun = true
	h.install.IncludeCRDs = true
	h.install.IsUpgrade = true // need it during dryRun to avoid missing helm labels - validation err
	h.install.Force = true
	return h.Install(ctx)
}

func (h *helmClient) GetKubeconfigPath() *string {
	return h.kubeconfigPath
}

func (h *helmClient) GetReleaseNamespace() string {
	return h.namespace
}

func (h *helmClient) GetClusterName() string {
	if h.clusterName != nil {
		return *h.clusterName
	}
	return ""
}

// prepareChartAndValues - loads the chart from the given local path and values specified
func (h *helmClient) prepareChartAndValues() (*chart.Chart, helmValues, error) {
	c, err := loader.Load(h.chartPath)
	if err != nil {
		return nil, nil, err
	}
	var vals helmValues
	if h.valuesPath != nil {
		valBytes, err := os.ReadFile(*h.valuesPath)
		if err != nil {
			return nil, nil, err
		}
		err = yaml.Unmarshal(valBytes, &vals)
		if err != nil {
			return nil, nil, err
		}
	} else {
		vals = h.values
	}

	return c, vals, nil
}

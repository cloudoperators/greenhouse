// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"context"
	"errors"
	"github.com/cloudoperators/greenhouse/pkg/internal/local/klient"
	"github.com/cloudoperators/greenhouse/pkg/internal/local/utils"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
)

type Options struct {
	ClusterName string
	ReleaseName string
	Namespace   string
	ChartPath   string
	ValuesPath  *string
}

type helmValues map[string]interface{}

type ClientOption func(*Options)

// WithChartPath - sets the chartPath flag for the helm client
// Note: Absolute paths are preferred
func WithChartPath(chartPath string) ClientOption {
	return func(h *Options) {
		h.ChartPath = chartPath
	}
}

// WithClusterName - sets the clusterName flag for the helm client
// used in a kind cluster scenario. By providing a kind cluster name,
// the kubeconfig will be fetched for the kind cluster using kind.getKubeCfg(clusterName, false)
func WithClusterName(clusterName string) ClientOption {
	return func(h *Options) {
		h.ClusterName = clusterName
	}
}

// WithReleaseName - sets the releaseName flag for the helm client
// release name will be used to install the chart or render the template with release labels
func WithReleaseName(releaseName string) ClientOption {
	return func(h *Options) {
		h.ReleaseName = releaseName
	}
}

// WithNamespace - sets the namespace flag for the helm client
// namespace will be used to install the chart or render the template
func WithNamespace(namespace string) ClientOption {
	return func(h *Options) {
		h.Namespace = namespace
	}
}

// WithValuesPath - sets the valuesPath flag for the helm client
// values provided in the file will be used to render the chart
// if no values path is provided, the default values will be used from util.GetManagerHelmValues()
func WithValuesPath(valuesPath string) ClientOption {
	return func(h *Options) {
		h.ValuesPath = utils.StringP(valuesPath)
	}
}

// apply - applies the Options to the client
func apply(options *Options) *client {
	return &client{
		clusterName: options.ClusterName,
		chartPath:   options.ChartPath,
		releaseName: options.ReleaseName,
		namespace:   options.Namespace,
		valuesPath:  options.ValuesPath,
	}
}

type client struct {
	install           *action.Install
	upgrade           *action.Upgrade
	clusterName       string
	releaseName       string
	namespace         string
	chartPath         string
	values            map[string]interface{}
	valuesPath        *string
	tmpKubeConfigPath string
}

// IHelm - interface wrapper for an actual Helm client
type IHelm interface {
	Install(ctx context.Context) (string, error)
	Template(ctx context.Context) (string, error)
}

// NewClient - creates a new Helm client with given ClientOption options
// currently supporting helm install and template actions and can be extended to support other actions
func NewClient(ctx context.Context, opts ...ClientOption) (IHelm, error) {
	logger := utils.NewKLog(ctx)
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}
	hc := apply(options)

	if hc.clusterName == "" {
		return nil, errors.New("cluster name must be provided")
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

	flags := &genericclioptions.ConfigFlags{
		Namespace: &hc.namespace,
	}

	configStr, err := klient.GetKubeCfg(hc.clusterName, false)
	if err != nil {
		return nil, err
	}
	tmpKubeConfigPath, err := utils.RandomWriteToTmpFolder(hc.clusterName, configStr)
	if err != nil {
		return nil, err
	}
	hc.tmpKubeConfigPath = tmpKubeConfigPath
	flags.KubeConfig = &tmpKubeConfigPath

	actionConfig := new(action.Configuration)
	err = actionConfig.Init(
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
func (c *client) Install(ctx context.Context) (string, error) {
	localChart, vals, err := c.prepareChartAndValues()
	if err != nil {
		return "", err
	}
	rel, err := c.install.RunWithContext(ctx, localChart, vals)
	if err != nil {
		return "", err
	}
	return rel.Manifest, nil
}

// Template - returns the rendered template of the chart
func (c *client) Template(ctx context.Context) (string, error) {
	c.install.ReleaseName = c.releaseName
	c.install.Namespace = c.namespace
	c.install.DryRun = true
	c.install.IncludeCRDs = true
	c.install.IsUpgrade = true // need it during dryRun to avoid missing helm labels - validation err
	c.install.Force = true
	defer utils.FileCleanUp(c.tmpKubeConfigPath)
	return c.Install(ctx)
}

// prepareChartAndValues - loads the chart from the given local path and values specified
func (c *client) prepareChartAndValues() (*chart.Chart, helmValues, error) {
	localChart, err := loader.Load(c.chartPath)
	if err != nil {
		return nil, nil, err
	}
	var vals helmValues
	if c.valuesPath != nil {
		valBytes, err := os.ReadFile(*c.valuesPath)
		if err != nil {
			return nil, nil, err
		}
		err = yaml.Unmarshal(valBytes, &vals)
		if err != nil {
			return nil, nil, err
		}
	} else {
		vals = c.values
	}
	return localChart, vals, nil
}

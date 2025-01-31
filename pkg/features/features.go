package features

import (
	"context"
	"errors"
	"sync"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

const (
	DexFeatureKey             = "dex"
	featureConfigMapName      = "greenhouse-feature-flags"
	featureConfigMapNamespace = "greenhouse"
)

type features struct {
	m         sync.Mutex
	k8sClient client.Client
	raw       map[string]string
	dex       *dexFeatures `yaml:"dex"`
}

type dexFeatures struct {
	Storage string `yaml:"storage"`
}

type Features interface {
	GetDexStorageType(ctx context.Context) *string
}

func NewFeatures(ctx context.Context, k8sClient client.Client) (Features, error) {
	featureMap := &corev1.ConfigMap{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: featureConfigMapName, Namespace: featureConfigMapNamespace}, featureMap); err != nil {
		return nil, err
	}
	return &features{
		k8sClient: k8sClient,
		raw:       featureMap.Data,
	}, nil
}

func (f *features) resolveDexFeatures() error {
	f.m.Lock()
	defer f.m.Unlock()

	// Extract the `dex` key from the ConfigMap
	dexRaw, exists := f.raw[DexFeatureKey]
	if !exists {
		return errors.New("dex feature not found in ConfigMap")
	}

	// Unmarshal the `dex` YAML string into the struct
	dex := &dexFeatures{}
	err := yaml.Unmarshal([]byte(dexRaw), dex)
	if err != nil {
		return err
	}

	f.dex = dex
	return nil
}

func (f *features) GetDexStorageType(ctx context.Context) *string {
	if f.dex != nil {
		return clientutil.Ptr(f.dex.Storage)
	}
	if err := f.resolveDexFeatures(); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "failed to resolve dex features")
		return clientutil.Ptr("")
	}
	if f.dex.Storage == "" {
		return nil
	}
	return clientutil.Ptr(f.dex.Storage)
}

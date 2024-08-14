package manifests

import (
	"context"
	"fmt"
	"github.com/cloudoperators/greenhouse/hack/localenv/pkg/klient"
	"github.com/cloudoperators/greenhouse/hack/localenv/pkg/utils"
	"github.com/vladimirvivien/gexe"
	"strings"
)

type manifests struct {
	hc           klient.IHelm
	excludeKinds []string
	crdOnly      bool
	webhook      *Webhook
}

type ISetup interface {
	Setup(ctx context.Context) error
}

type IManifest interface {
	GenerateManifests(ctx context.Context) ([]map[string]interface{}, error)
	ApplyManifests(resources []map[string]interface{}) error
	SetupWebhookManifest(resources []map[string]interface{}) (map[string]interface{}, error)
}

func NewManifestsSetup(hc klient.IHelm, webhook *Webhook, excludeKinds []string, crdOnly bool) ISetup {
	return &manifests{hc: hc, webhook: webhook, excludeKinds: excludeKinds, crdOnly: crdOnly}
}

func NewCmdManifests(hc klient.IHelm, excludeKinds []string, crdOnly bool) IManifest {
	return &manifests{hc: hc, excludeKinds: excludeKinds, crdOnly: crdOnly}
}

func (m *manifests) Setup(ctx context.Context) error {
	resources, err := m.generateAllManifests(ctx)
	if err != nil {
		return err
	}
	excluded := m.resourceExclusion(resources)
	filtered := m.filterCustomResources(excluded)
	if m.webhook == nil {
		utils.Log("no webhook configuration provided, skipping webhook kustomization")
		noWbManifests := excludeResources(filtered, []string{"MutatingWebhookConfiguration", "ValidatingWebhookConfiguration"})
		return m.ApplyManifests(noWbManifests)
	}
	webHookManifest, err := m.SetupWebhookManifest(resources)
	if err != nil {
		return err
	}
	filtered = append(filtered, webHookManifest)
	return m.ApplyManifests(filtered)
}

func (m *manifests) GenerateManifests(ctx context.Context) ([]map[string]interface{}, error) {
	resources, err := m.generateAllManifests(ctx)
	if err != nil {
		return nil, err
	}
	excluded := m.resourceExclusion(resources)
	return m.filterCustomResources(excluded), nil
}

func (m *manifests) ApplyManifests(resources []map[string]interface{}) error {
	manifests, err := utils.Stringify(resources)
	if err != nil {
		return err
	}
	return m.apply(manifests)
}

func (m *manifests) apply(manifests string) error {
	var cmd string
	var exec = gexe.New()
	tmpResourcePath, err := utils.WriteToTmpFolder("kind-resources", manifests)
	if err != nil {
		return err
	}
	kubeconfigPath := m.hc.GetKubeconfigPath()
	releaseNamespace := m.hc.GetReleaseNamespace()
	if kubeconfigPath != nil {
		cmd = fmt.Sprintf("kubectl apply --kubeconfig=%s -f %s -n %s", *kubeconfigPath, tmpResourcePath, releaseNamespace)
	} else {
		cmd = fmt.Sprintf("kubectl apply -f %s -n %s", tmpResourcePath, releaseNamespace)
	}
	defer func() {
		tmpFiles := []string{tmpResourcePath}
		if kubeconfigPath != nil {
			tmpFiles = append(tmpFiles, *kubeconfigPath)
		}
		cleanUp(tmpFiles...)
	}()
	proc := exec.RunProc(cmd)
	if err := proc.Err(); err != nil {
		return fmt.Errorf("failed to apply manifests: %s - %w", proc.Result(), err)
	}
	utils.Logf("%s", proc.Result())
	return nil
}

func (m *manifests) generateAllManifests(ctx context.Context) ([]map[string]interface{}, error) {
	templates, err := m.hc.Template(ctx)
	if err != nil {
		return nil, err
	}
	docs := strings.Split(templates, "---")
	resources := make([]map[string]interface{}, 0)
	for _, doc := range docs {
		resource, err := utils.RawK8sInterface([]byte(doc))
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
		}
		if resource != nil {
			resources = append(resources, resource)
		}
	}
	return resources, nil
}

func (m *manifests) resourceExclusion(resources []map[string]interface{}) []map[string]interface{} {
	if len(m.excludeKinds) == 0 {
		return resources
	}
	return excludeResources(resources, m.excludeKinds)
}

func excludeResources(resources []map[string]interface{}, exclusions []string) []map[string]interface{} {
	excludeResources := make([]map[string]interface{}, 0)
	excludeResources = append(excludeResources, resources...)
	for i := 0; i < len(excludeResources); {
		if k, ok := excludeResources[i]["kind"].(string); ok && utils.SliceContains(exclusions, k) {
			excludeResources[i] = excludeResources[len(excludeResources)-1]
			excludeResources = excludeResources[:len(excludeResources)-1]
		} else {
			i++
		}
	}
	return excludeResources
}

func (m *manifests) filterCustomResources(resources []map[string]interface{}) []map[string]interface{} {
	if m.crdOnly {
		return filterResourcesBy(resources, "CustomResourceDefinition")
	}
	return resources
}

func filterResourcesBy(resources []map[string]interface{}, filterBy string) []map[string]interface{} {
	filteredResource := make([]map[string]interface{}, 0)
	filteredResource = append(filteredResource, resources...)
	for i := 0; i < len(filteredResource); {
		if k, ok := filteredResource[i]["kind"].(string); ok && k != filterBy {
			filteredResource[i] = filteredResource[len(filteredResource)-1]
			filteredResource = filteredResource[:len(filteredResource)-1]
		} else {
			i++
		}
	}
	return filteredResource
}

func cleanUp(files ...string) {
	// clean up the tmp files
	for _, file := range files {
		if err := utils.RemoveTmpFile(file); err != nil {
			utils.LogErr("warning: tmp file deletion: %s", err)
		}
	}
}

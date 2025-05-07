package flux

import (
	"strings"

	sourcecontroller "github.com/fluxcd/source-controller/api/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

func convertName(repoName string) (convertedName string, repoType string) {
	repoType = sourcecontroller.HelmRepositoryTypeDefault
	// set the helm repository type to OCI if the repo name starts with oci://
	if strings.HasPrefix(repoName, "oci://") {
		repoType = sourcecontroller.HelmRepositoryTypeOCI
	}
	// remove prefixes
	var prefixes = []string{
		"oci://",
		"https://",
		"http://",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(repoName, prefix) {
			convertedName = strings.TrimPrefix(repoName, prefix)
			break
		}
	}

	convertedName = strings.ReplaceAll(convertedName, ".", "-")
	convertedName = strings.ReplaceAll(convertedName, "/", "-")
	return convertedName, repoType
}

func generateChartName(plDf *greenhousev1alpha1.PluginDefinition) string {
	return strings.Join([]string{plDf.Spec.HelmChart.Name, plDf.Spec.HelmChart.Version}, "-")
}

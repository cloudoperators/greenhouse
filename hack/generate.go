package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

var (
	sourceCRDDir string = "./hack/crd"
	targetDir    string = "./charts/manager/crds"
)

func main() {
	flag.StringVar(&sourceCRDDir, "crd-dir", "./hack/crd", "source directory for crd bases and patches")
	flag.StringVar(&targetDir, "charts-crd-dir", "./charts/manager/crds", "target directory for the rendered crds")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if _, err := os.Stat(filepath.Join(sourceCRDDir, "bases")); os.IsNotExist(err) {
		slog.Info("source directory does not exist, nothing to do")
		return
	}

	crdBases, err := filepath.Glob(filepath.Join(sourceCRDDir, "bases", "*.yaml"))
	if err != nil {
		log.Fatalf("failed to glob crd yamls: %s", err.Error())
	}
	if len(crdBases) == 0 {
		slog.Info("no crds found, nothing to do")
		return
	}

	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		log.Fatalf("failed to create directory: %s", err.Error())
	}

	for _, file := range crdBases {
		destFile := filepath.Join(targetDir, filepath.Base(file))
		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("failed to read file: %s", err.Error())
		}
		contentStr := string(content)

		group, kind := groupKindFromFile(file)
		conversionPatch, err := extractConversionPatch(group, kind)
		if err != nil {
			log.Fatalf("failed to extract conversion patch: %s", err.Error())
		}
		conversionSpec := extractConversionSpec(conversionPatch)

		if conversionSpec != "" {
			contentStr = insertCRDConversionSpec(contentStr, conversionSpec)
		}

		if err = os.MkdirAll(filepath.Dir(destFile), os.ModePerm); err != nil {
			log.Fatalf("failed to create destination file directory: %s", err.Error())
		}
		if err = os.WriteFile(destFile, []byte(contentStr), os.ModePerm); err != nil {
			log.Fatalf("failed to write file: %s", err.Error())
		}
		log.Printf("CRD %s written to %s", file, destFile)
	}
}

// groupKindFromFile returns the group and version of a CRD file in the format <group>.<domain>_<kind>.yaml
func groupKindFromFile(file string) (group, kind string) {
	splits := strings.Split(filepath.Base(file), "_")
	if len(splits) == 2 {
		group = strings.Split(splits[0], ".")[0]
		kind = strings.TrimSuffix(splits[1], ".yaml")
	}
	return group, kind
}

// extractConversionPatch returns the conversion webhook patch for a given group and kind
// first it looks for files in the format webhook_<group>_<kind>.yaml
// if no files are found it looks for files in the format webhook_<kind>.yaml
// if no files are found it returns an empty string
func extractConversionPatch(group, kind string) (string, error) {
	groupKindWebhookPattern := filepath.Join(sourceCRDDir, "patches", fmt.Sprintf("webhook_*%s*_*%s*.yaml", group, kind))
	patches, err := filepath.Glob(groupKindWebhookPattern)
	if err != nil {
		return "", err
	}
	if len(patches) == 0 {
		kindWebhookPattern := filepath.Join(sourceCRDDir, "patches", fmt.Sprintf("webhook_*%s*.yaml", kind))
		patches, err = filepath.Glob(kindWebhookPattern)
		if err != nil {
			return "", fmt.Errorf("failed to list patches: %w", err)
		}
	}
	if len(patches) > 0 {
		patchContent, err := os.ReadFile(patches[0])
		if err != nil {
			return "", fmt.Errorf("failed to read patch file: %w", err)
		}
		return string(patchContent), nil
	}
	return "", nil
}

// extractConversionSpec returns the conversion spec from a given patch
// if no conversion spec is found it returns an empty string
// it returns everything after the first occurrence of the string "conversion:"
func extractConversionSpec(patch string) string {
	startIndex := strings.Index(patch, "conversion:")
	if startIndex == -1 {
		return ""
	}
	return patch[startIndex:]
}

// insertCRDConversionSpec inserts a conversion spec into a CRD file
// it looks for the first occurrence of the string "spec:" and inserts the conversion spec after it
// if "spec:" is not found it returns the original content
func insertCRDConversionSpec(content, conversionSpec string) string {
	specIndex := strings.Index(content, "spec:")
	if specIndex == -1 {
		return content
	}
	return content[:specIndex+5] + "\n  " + strings.TrimRight(conversionSpec, "\n") + content[specIndex+5:]
}

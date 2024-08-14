package utils

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"
	"io"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/klog/v2"
	"os"
	"slices"
	"strings"
)

func Log(args ...any) {
	args[0] = "===== 🤖 " + args[0].(string)
	klog.Info(args...)
}

func Logf(format string, args ...any) {
	klog.Infof("===== 🤖 "+format, args...)
}

func LogErr(format string, args ...any) {
	klog.Infof("===== 😵 "+format, args...)
}

func NewKLog(ctx context.Context) logr.Logger {
	return klog.FromContext(ctx)
}

func StringP(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

// GetManagerHelmValues - returns the default values for the manager helm chart
func GetManagerHelmValues() map[string]interface{} {
	return map[string]interface{}{
		"alerts": map[string]interface{}{
			"enabled": false,
		},
		"global": map[string]interface{}{
			"dnsDomain": "localhost",
		},
	}
}

func SliceContains(slice []string, item string) bool {
	return slices.ContainsFunc(slice, func(s string) bool {
		return strings.ToLower(s) == strings.ToLower(item)
	})
}

// WriteToTmpFolder - writes the provided content to temp folder in OS
func WriteToTmpFolder(fileName, content string) (string, error) {
	file, err := os.CreateTemp("", fmt.Sprintf("kind-cluster-%s", fileName))
	if err != nil {
		return "", err
	}
	defer file.Close()
	if n, err := io.WriteString(file, content); n == 0 || err != nil {
		return "", fmt.Errorf("kind kubecfg file: bytes copied: %d: %w]", n, err)
	}
	return file.Name(), nil
}

func RemoveTmpFile(file string) error {
	if err := os.Remove(file); err != nil {
		return fmt.Errorf("failed to remove file %s: %w", file, err)
	}
	return nil
}

func RawK8sInterface(yamlBytes []byte) (map[string]interface{}, error) {
	var data map[string]interface{}
	err := kyaml.Unmarshal(yamlBytes, &data)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling YAML bytes: %w", err)
	}
	return data, nil
}

func Stringy(data map[string]interface{}) (string, error) {
	s, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

func Stringify(data []map[string]interface{}) (string, error) {
	stringSources := make([]string, 0)
	for _, d := range data {
		s, err := Stringy(d)
		if err != nil {
			return "", err
		}
		stringSources = append(stringSources, s)
	}
	return strings.Join(stringSources, "\n---\n"), nil
}

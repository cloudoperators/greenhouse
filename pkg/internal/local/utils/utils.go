// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"slices"
	"strings"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	syaml "sigs.k8s.io/yaml"
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

func Int32P(i int32) *int32 {
	return &i
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
		return strings.EqualFold(s, item)
	})
}

func WriteToPath(dir, fileName, content string) error {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	filePath := dir + "/" + fileName
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			LogErr("failed to close file %s after write: %s", filePath, err.Error())
		}
	}(file)
	if n, err := io.WriteString(file, content); n == 0 || err != nil {
		return fmt.Errorf("error writing file %s: %w", file.Name(), err)
	}
	return nil
}

// RandomWriteToTmpFolder - writes the provided content to temp folder in OS
// Concurrent writes do not conflict as the file name is appended with a random string
func RandomWriteToTmpFolder(fileName, content string) (string, error) {
	file, err := os.CreateTemp("", "kind-cluster-"+fileName)
	if err != nil {
		return "", err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			LogErr("failed to close file %s after write: %s", fileName, err.Error())
		}
	}(file)
	if n, err := io.WriteString(file, content); n == 0 || err != nil {
		return "", fmt.Errorf("kind kubecfg file: bytes copied: %d: %w]", n, err)
	}
	return file.Name(), nil
}

// RawK8sInterface - unmarshalls the provided YAML bytes into a map[string]interface{}
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

// FromYamlToK8sObject - Converts a YAML document to a Kubernetes object
// if yaml contains multiple documents, then corresponding kubernetes objects should be provided
func FromYamlToK8sObject(doc string, resources ...any) error {
	yamlBytes := []byte(doc)
	dec := kyaml.NewDocumentDecoder(io.NopCloser(bytes.NewReader(yamlBytes)))
	buffer := make([]byte, len(yamlBytes))

	for _, resource := range resources {
		n, err := dec.Read(buffer)
		if err != nil {
			return err
		}
		err = kyaml.Unmarshal(buffer[:n], resource)
		if err != nil {
			return err
		}
	}
	return nil
}

// FromK8sObjectToYaml - Converts a Kubernetes object to a YAML document
func FromK8sObjectToYaml(obj client.Object, gvk schema.GroupVersion) ([]byte, error) {
	scheme := kruntime.NewScheme()
	err := clientgoscheme.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	codec := serializer.NewCodecFactory(scheme).LegacyCodec(gvk)
	jsonBytes, err := kruntime.Encode(codec, obj)
	if err != nil {
		return nil, fmt.Errorf("error encoding object to JSON bytes: %w", err)
	}

	yamlBytes, err := syaml.JSONToYAML(jsonBytes)
	if err != nil {
		return nil, fmt.Errorf("error marshalling as YAML bytes: %w", err)
	}

	return yamlBytes, nil
}

func CheckIfFileExists(f string) bool {
	_, err := os.Stat(f)
	return !os.IsNotExist(err)
}

func CleanUp(files ...string) {
	// clean up the tmp files
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			LogErr("failed to remove file %s: %s", file, err.Error())
		}
	}
}

func GetHostPlatform() string {
	var platform string
	switch runtime.GOARCH {
	case "amd64":
		platform = "linux/amd64"
	case "arm64":
		platform = "linux/arm64"
	default:
		platform = "linux/amd64"
	}
	return platform
}

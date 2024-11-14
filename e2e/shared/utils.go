// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package shared

import (
	"bytes"
	"io"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/klog/v2"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/pkg/apis/greenhouse/v1alpha1"
)

func Log(args ...any) {
	args[0] = "===== 🤖 " + args[0].(string) //nolint:errcheck
	klog.InfoDepth(1, args...)
}

func Logf(format string, args ...any) {
	klog.InfofDepth(1, "===== 🤖 "+format, args...)
}

func LogErr(format string, args ...any) {
	klog.InfofDepth(1, "===== 😵 "+format, args...)
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

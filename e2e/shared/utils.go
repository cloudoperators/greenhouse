package shared

import (
	"bytes"
	"io"
	"os"

	kyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/klog/v2"
)

func Log(args ...any) {
	args[0] = "===== 🤖 " + args[0].(string)
	klog.InfoDepth(1, args...)
}

func Logf(format string, args ...any) {
	klog.InfofDepth(1, "===== 🤖 "+format, args...)
}

func CheckError(err error) {
	if err != nil {
		klog.ErrorfDepth(1, "===== 😵 error: %s", err)
		os.Exit(-1)
	}
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

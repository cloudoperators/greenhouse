package shared

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewKubeClientFromConfigWithScheme(configStr string, userScheme ...func(s *runtime.Scheme) error) (*rest.Config, client.Client, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(configStr))
	if err != nil {
		return nil, nil, err
	}
	if len(userScheme) > 0 {
		schemeBuilder := runtime.SchemeBuilder(userScheme)
		utilruntime.Must(schemeBuilder.AddToScheme(scheme.Scheme))
	}
	k8sClient, err := client.New(config, client.Options{Scheme: scheme.Scheme})
	return config, k8sClient, err
}

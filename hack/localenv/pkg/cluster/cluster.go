package cluster

import (
	"github.com/cloudoperators/greenhouse/hack/localenv/pkg/kind"
	"github.com/cloudoperators/greenhouse/hack/localenv/pkg/utils"
)

type Cluster struct {
	Name       string  `json:"name"`
	Namespace  *string `json:"namespace"`
	skipCreate bool
}

type IClusterSetup interface {
	Setup() error
}

func NewCmdCluster(name, namespaceName string) IClusterSetup {
	return &Cluster{
		Name:      name,
		Namespace: utils.StringP(namespaceName),
	}
}

func (c *Cluster) Setup() error {
	if c.skipCreate {
		return nil
	}
	err := kind.CreateKindCluster(c.Name)
	if err != nil {
		return err
	}
	if c.Namespace == nil {
		return nil
	}
	return kind.CreateNamespace(*c.Namespace)
}

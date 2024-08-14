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
	Delete() error
	List() error
}

func NewCmdCluster(name, namespaceName string) IClusterSetup {
	return &Cluster{
		Name:      name,
		Namespace: utils.StringP(namespaceName),
	}
}

func Configure(config *Cluster, skipCreate bool) *Cluster {
	config.skipCreate = skipCreate
	return config
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

func (c *Cluster) Delete() error {
	return kind.DeleteCluster(c.Name)
}

func (c *Cluster) List() error {
	clusters, err := kind.GetClusters()
	if err != nil {
		return err
	}
	for _, c := range clusters {
		utils.Logf("cluster: %s", c)
	}
	return nil
}

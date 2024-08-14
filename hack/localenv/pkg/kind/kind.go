package kind

import (
	"fmt"
	"github.com/cloudoperators/greenhouse/hack/localenv/pkg/utils"
	"github.com/vladimirvivien/gexe"
	"strings"
)

var exec = gexe.New()

func CreateKindCluster(clusterName string) error {
	exists, err := ClusterExists(clusterName)
	if err != nil {
		return err
	} else if exists {
		utils.Logf("kind cluster with name %s already exists", clusterName)
		exec.SetVar("name", clusterName)
		proc := exec.RunProc("kubectl config set-context kind-${name}")
		if err := proc.Err(); err != nil {
			return err
		}
		utils.Logf("%s", proc.Result())
		return nil
	}

	exec.SetVar("name", clusterName)
	proc := exec.RunProc("kind create cluster --name ${name}")
	if err := proc.Err(); err != nil {
		return fmt.Errorf("failed to create kind cluster: %w", err)
	}
	utils.Logf("%s", proc.Result())
	utils.Log("cluster is ready 🚀")
	return nil
}

func DeleteCluster(clusterName string) error {
	exists, err := ClusterExists(clusterName)
	if err != nil {
		return err
	} else if !exists {
		utils.Logf("kind cluster with name %s does not exist", clusterName)
		return nil
	}
	exec.SetVar("name", clusterName)
	proc := exec.RunProc("kind delete cluster --name ${name}")
	if err := proc.Err(); err != nil {
		return err
	}
	utils.Logf("%s", proc.Result())
	utils.Logf("kind cluster with name %s deleted", clusterName)
	return nil
}

func ClusterExists(clusterName string) (bool, error) {
	clusters, err := GetClusters()
	if err != nil {
		return false, fmt.Errorf("failed to check if cluster exists: %w", err)
	}
	utils.Logf("checking if cluster %s exists...", clusterName)
	for _, c := range clusters {
		if c == clusterName {
			return true, nil
		}
	}
	return false, nil
}

func GetClusters() ([]string, error) {
	proc := exec.RunProc("kind get clusters")
	if err := proc.Err(); err != nil {
		return nil, err
	}
	return strings.FieldsFunc(proc.Result(), func(r rune) bool {
		return r == '\n'
	}), nil
}

func CreateNamespace(namespaceName string) error {
	if strings.TrimSpace(namespaceName) == "" {
		return nil
	}
	utils.Logf("creating namespaceName %s", namespaceName)
	errs := make([]string, 0)
	exec.SetVar("namespace", namespaceName)
	pipe := exec.Pipe("kubectl create namespace ${namespace} --dry-run=client -o yaml", "kubectl apply -f -")
	for _, p := range pipe.Procs() {
		if err := p.Err(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to create namespace: %s", strings.Join(errs, ", "))
	}
	utils.Logf("%s", pipe.LastProc().Result())
	return nil
}

func GetKubeCfg(clusterName string, internal bool) (string, error) {
	cmd := fmt.Sprintf("kind get kubeconfig --name %s", clusterName)
	if internal {
		cmd += " --internal"
	}
	proc := exec.RunProc(cmd)
	if err := proc.Err(); err != nil {
		return "", fmt.Errorf("failed to get kubeconfig: %w", err)
	}
	return proc.Result(), nil
}

func LoadImage(image string, clusterName string) error {
	exec.SetVar("image", image)
	exec.SetVar("cluster", clusterName)
	utils.Logf("loading docker image %s into %s cluster", image, clusterName)
	cmd := exec.RunProc("kind load docker-image ${image} --name ${cluster}")
	if err := cmd.Err(); err != nil {
		return err
	}
	utils.Logf("%s", cmd.Result())
	return nil
}

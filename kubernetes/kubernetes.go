package kubernetes

import (
	"context"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"gopkg.in/alecthomas/kingpin.v2"

	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

type K8s struct {
	clientset *client.Clientset
	config    *rest.Config
	log       *zap.Logger
}

// type K8SUtil interface {
// 	getNodeCondition(nodeName string) []core.NodeCondition
// 	changeNodeCondition(nodeName string, conds []core.NodeCondition) (err error)
// }

func NewClient(log *zap.Logger, apiserver *string, kubecfg *string) *K8s {
	config, err := BuildConfigFromFlags(*apiserver, *kubecfg)
	kingpin.FatalIfError(err, "cannot create Kubernetes client configuration")
	clientset, err := client.NewForConfig(config)
	kingpin.FatalIfError(err, "cannot create Kubernetes client")

	return &K8s{config: config, clientset: clientset, log: log}
}

func BuildConfigFromFlags(apiserver, kubecfg string) (*rest.Config, error) {
	if home := homedir.HomeDir(); kubecfg == "" && home != "" {
		filePath := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			kubecfg = filePath
		}
	}

	if kubecfg != "" || apiserver != "" {
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubecfg},
			&clientcmd.ConfigOverrides{ClusterInfo: api.Cluster{Server: apiserver}}).ClientConfig()
	}
	return rest.InClusterConfig()
}

func (k *K8s) GetNodeCondition(nodeName string) []core.NodeCondition {
	node, _ := k.clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, meta.GetOptions{})

	return node.Status.Conditions
}

func (k *K8s) ChangeNodeCondition(nodeName string, conds []core.NodeCondition) (err error) {
	// Refresh the node object
	freshNode, err := k.clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, meta.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		return nil
	}

	conditionUpdated := false

	for i, condition := range freshNode.Status.Conditions {
		for _, newCond := range conds {
			if condition.Type == newCond.Type {
				freshNode.Status.Conditions[i].LastHeartbeatTime = newCond.LastHeartbeatTime
				freshNode.Status.Conditions[i].LastTransitionTime = newCond.LastTransitionTime
				freshNode.Status.Conditions[i].Message = newCond.Message
				freshNode.Status.Conditions[i].Status = newCond.Status
				conditionUpdated = true
				break
			}
		}
	}

	if !conditionUpdated { // There was no condition found, let's create one
		freshNode.Status.Conditions = append(freshNode.Status.Conditions, conds...)
	}

	for _, c := range freshNode.Status.Conditions {
		println(c.Type, c.Status)
	}

	if _, err := k.clientset.CoreV1().Nodes().UpdateStatus(context.TODO(), freshNode, meta.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}

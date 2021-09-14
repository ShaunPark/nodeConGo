package main

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"gopkg.in/alecthomas/kingpin.v2"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ShaunPark/nodeConGo/kubernetes"
)

var (
	kubecfg   = kingpin.Flag("kubeconfig", "Path to kubeconfig file. Leave unset to use in-cluster config.").String()
	apiserver = kingpin.Flag("master", "Address of Kubernetes API server. Leave unset to use in-cluster config.").String()

	changeCommand = kingpin.Command("change", "Change Condition Status")
	deleteCommand = kingpin.Command("delete", "Delete Condition")

	conditionType = kingpin.Flag("condition", "(Optional) Omit if you want to change all conditions except 'Ready'").Short('t').String()
	nodeName      = kingpin.Flag("nodename", "node name to change condition").Short('n').Required().String()

	status  = changeCommand.Flag("status", "condition type").Short('s').Required().String()
	message = changeCommand.Flag("message", "(Optional) Message of status").Short('m').String()

	validStatus       = []core.ConditionStatus{core.ConditionFalse, core.ConditionTrue}
	inValidConditions = []string{"Ready", "KernelDeadLock", "CorruptDockerOverlay2", "ReadonlyFilesystem", "NetworkUnavailable", "DiskPressure", "MemoryPressure", "PIDPressure", "EFSConnectFail"}
)

func main() {

	switch kingpin.Parse() {
	case changeCommand.FullCommand():
		change()
	case deleteCommand.FullCommand():
		delete()
	}
}

func change() {
	log, err := zap.NewProduction()
	kingpin.FatalIfError(err, "cannot create log")

	_status := core.ConditionStatus(*status)
	_conditionType := core.NodeConditionType(*conditionType)

	if !contains(validStatus, *status) {
		kingpin.FatalUsage("Invalid status %s. Only 'True' and 'False' are avaliable", *status)
	}
	if _status == core.ConditionTrue && len(_conditionType) == 0 {
		kingpin.FatalUsage("Changing all conditions to 'True' is not allowed")
	}
	if _status == core.ConditionFalse && _conditionType == "Ready" {
		kingpin.FatalUsage("Changing the status of 'Ready' condition to 'False' is not allowed.")
	}

	do(*nodeName, _status, _conditionType, *message, log)
}

func delete() {
	log, err := zap.NewProduction()
	kingpin.FatalIfError(err, "cannot create log")

	if containStr(inValidConditions, *conditionType) {
		kingpin.FatalUsage("Delete condition '%s' is not allowed.", *conditionType)
	}

	k := kubernetes.NewClient(log, apiserver, kubecfg)

	hasCondition := false
	if conditions, err := k.GetNodeConditions(*nodeName); err == nil {
		newConditions := make([]core.NodeCondition, 0)

		for _, condition := range conditions {
			if condition.Type == core.NodeConditionType(*conditionType) {
				hasCondition = true
			} else {
				fmt.Printf("%s\n", condition.Type)
				newConditions = append(newConditions, condition)
			}
		}

		if hasCondition {
			k.PatchNodeStatus(*nodeName, newConditions)
		}
	}
}

var _okMessage = map[string]string{
	"KernelDeadLock":        "kernel has no deadlock",
	"CorruptDockerOverlay2": "docker overlay2 is functioning properly",
	"ReadonlyFilesystem":    "Filesystem is not read-only",
	"NetworkUnavailable":    "Weave pod has set this",
	"DiskPressure":          "kubelet has no disk pressure",
	"MemoryPressure":        "kubelet has sufficient memory available",
	"PIDPressure":           "kubelet has sufficient PID available",
	"EFSConnectFail":        "EFS is mounted successfully",
}

func contains(s []core.ConditionStatus, substr string) bool {
	for _, v := range s {
		if v == core.ConditionStatus(substr) {
			return true
		}
	}
	return false
}

func containStr(s []string, substr string) bool {
	for _, v := range s {
		if v == substr {
			return true
		}
	}
	return false
}

func filter(conds []core.NodeCondition, f func(cond core.NodeCondition) bool) *[]core.NodeCondition {
	filtered := []core.NodeCondition{}

	for _, c := range conds {
		if f(c) {
			filtered = append(filtered, c)
		}
	}
	return &filtered
}

type Condition struct {
	Status             core.ConditionStatus
	Type               core.NodeConditionType
	LastTransitionTime meta.Time
}

func do(nodeName string, status core.ConditionStatus, conditionType core.NodeConditionType, message string, log *zap.Logger) {
	k := kubernetes.NewClient(log, apiserver, kubecfg)

	conditions := k.GetNodeCondition(nodeName)
	var filtered []core.NodeCondition

	if len(conditionType) == 0 {
		log.Sugar().Infof("Change all node condition's status to '%s'", status)
		filtered = *filter(conditions, func(cond core.NodeCondition) bool { return cond.Type != "Ready" })
	} else {
		log.Sugar().Infof("Change node status of condition '%s' to '%s", conditionType, status)
		cType := conditionType
		filtered = *filter(conditions, func(cond core.NodeCondition) bool { return cond.Type == cType })
	}

	for i, c := range filtered {
		if c.Status != status {
			filtered[i].LastTransitionTime = meta.Time{Time: time.Now()}
		}

		if len(message) != 0 {
			filtered[i].Message = message
		} else {
			msg := _okMessage[string(conditionType)]
			if status == core.ConditionFalse && len(msg) != 0 {
				filtered[i].Message = msg
			}
		}

		filtered[i].Status = status
	}

	if len(filtered) == 0 && len(conditionType) != 0 {
		now := meta.Time{Time: time.Now()}

		condition := core.NodeCondition{
			Type:               conditionType,
			Status:             status,
			Message:            message,
			Reason:             string(conditionType),
			LastHeartbeatTime:  now,
			LastTransitionTime: now,
		}
		filtered = append(filtered, condition)
	}

	if err := k.ChangeNodeCondition(nodeName, filtered); err != nil {
		log.Error(err.Error())
	}
}

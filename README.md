# Kubernetes Node Condition Changer
- 쿠버네티스의 노드 컨디션 상태를 변경하는 유틸리티
- 변경하고자 하는 컨디션이 없으면 해당 컨디션을 추가함
- 컨디션 타입이 정의되지 않으면 모든 컨디션의 상태를 일괄 변경. 단, True로 일괄변경은 불가.

## 사용법
```
usage: nodeConGo --nodename=NODENAME --status=STATUS [<flags>]

Flags:
      --help                   Show context-sensitive help (also try --help-long and --help-man).
      --kubeconfig=KUBECONFIG  Path to kubeconfig file. Leave unset to use in-cluster config or to use "~/.kube.config" .
      --master=MASTER          Address of Kubernetes API server. Leave unset to use in-cluster config or to use "~/.kube.config".
  -n, --nodename=NODENAME      node name to change condition
  -s, --status=STATUS          condition type
  -t, --condition=CONDITION    (Optional) Omit if you want to change all conditions except 'Ready'
  -m, --message=MESSAGE        (Optional) Message of status
```



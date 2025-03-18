#/bin/bash
set -vx
kubectl delete -n telegraf daemonset telegraf
kubectl delete -n telegraf configmap telegraf-config
kubectl delete -n telegraf clusterrolebinding list-nodes-rolebinding
kubectl delete -n telegraf clusterrole list-nodes-role
kubectl delete -n telegraf serviceaccount list-nodes-sa
kubectl apply -f telegraf-k8s-globaltag-injector-poc.yaml -n telegraf
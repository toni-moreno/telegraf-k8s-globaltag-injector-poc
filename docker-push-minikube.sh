eval $(minikube docker-env)
docker build -t k8s-node-label-extractor:latest .

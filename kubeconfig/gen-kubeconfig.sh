#!/bin/sh
set -e

TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
CA=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt
NS=$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace)
APISERVER="https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}"

cat <<EOF > /kube/config
apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority: ${CA}
    server: ${APISERVER}
  name: in-cluster
contexts:
- context:
    cluster: in-cluster
    namespace: ${NS}
    user: sa
  name: in-cluster
current-context: in-cluster
users:
- name: sa
  user:
    token: ${TOKEN}
EOF

echo "✅ kubeconfig written to /kube/config"

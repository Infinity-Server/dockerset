FROM bitnami/kubectl:latest
COPY gen-kubeconfig.sh /usr/local/bin/gen-kubeconfig
WORKDIR /kube
ENTRYPOINT ["/usr/local/bin/gen-kubeconfig"]

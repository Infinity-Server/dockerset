FROM alpine

ARG K3S_VERSION=v1.22.7+k3s1

RUN mkdir -p /var/lib/rancher/k3s && \
    mkdir -p /var/lib/kubelet && \
    mkdir -p /var/lib/cni && \
    mkdir -p /var/log && \
    mkdir -p /etc/rancher && \
    mkdir -p /host && \
    chmod 1777 /tmp && \
    apk add jq curl bash cifs-utils util-linux && \
    curl -fsSL https://github.com/k3s-io/k3s/releases/download/${K3S_VERSION}/k3s -o /bin/k3s && \
    chmod 755 /bin/k3s && \
    ln -s /bin/k3s /bin/kubectl && \
    ln -s /bin/k3s /bin/crictl && \
    ln -s /bin/k3s /bin/ctr && \
    mkdir -p /usr/libexec/kubernetes/kubelet-plugins/volume/exec/fstab~cifs && \
    curl -fsSL https://raw.githubusercontent.com/fstab/cifs/master/cifs -o /usr/libexec/kubernetes/kubelet-plugins/volume/exec/fstab~cifs/cifs && \
    chmod 755 /usr/libexec/kubernetes/kubelet-plugins/volume/exec/fstab~cifs/cifs

ADD init /init
ENTRYPOINT ["/init"]
CMD ["agent"]

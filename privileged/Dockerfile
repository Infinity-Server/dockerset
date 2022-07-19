FROM debian

WORKDIR /uml
ADD init /uml/init
ADD privileged /uml/privileged
RUN DEBIAN_FRONTEND=noninteractive apt-get -y update && \
    DEBIAN_FRONTEND=noninteractive apt-get -y install \
      tini screen iproute2 net-tools user-mode-linux uml-utilities linux-image-amd64 && \
    mv /usr/lib/uml/modules /uml/rootfs/lib/modules && \
    ln -s /uml/privileged /bin/privileged && \
    ln -s /uml/privileged /bin/PRIVILEGED && \
    passwd -d root

CMD ["/bin/sh"]

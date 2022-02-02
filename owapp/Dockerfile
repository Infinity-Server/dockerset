FROM ubuntu:20.04

ARG URL="https://cdn.ionewu.com/upgrade/d/DSM6_owapp-ARCH-2.3.21-6119.spk"

WORKDIR /work
RUN DEBIAN_FRONTEND=noninteractive apt-get -y update && \
    DEBIAN_FRONTEND=noninteractive apt-get -y install curl wget dpkg-dev xz-utils openssl iproute2
ADD download /work/download
RUN mkdir -p /volume1/OWDownload && \
    /work/download "${URL}" && \
    rm -rvf /work

WORKDIR /root
ADD init /root/init

EXPOSE 8283
VOLUME ["/volume1/OWDownload"]
ENTRYPOINT ["/root/init"]

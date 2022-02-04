FROM golang:1.17.6 AS builder
WORKDIR /go/src
COPY xunlei_cgi /go/src/xunlei_cgi
WORKDIR /go/src/xunlei_cgi
RUN go env -w GO111MODULE=on && go env -w GOPROXY=https://goproxy.cn,direct
RUN CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -v
RUN ls -alth 

FROM ubuntu:20.04 as rootfs

FROM ubuntu:20.04
ARG VERSION=2.2.3
WORKDIR /work
RUN DEBIAN_FRONTEND=noninteractive apt-get -y update && \
    DEBIAN_FRONTEND=noninteractive apt-get -y install curl wget xz-utils dpkg-dev
COPY --from=rootfs / /rootfs
RUN mkdir -p /rootfs/var/packages/pan-xunlei-com/shares && \
    mkdir -p /rootfs/var/packages/pan-xunlei-com/target && \
    curl -L https://github.com/Infinity-Server/docker_image_set/raw/blob/xunlei_cgi/v${VERSION}-DSM7.x-$(dpkg-architecture -q DEB_BUILD_ARCH).spk --output /work/xunlei.spk && \
    tar -xvf xunlei.spk && \
    tar -xvf package.tgz && \
    cp -rvf /work/bin/bin/* /rootfs/var/packages/pan-xunlei-com/target/ && \
    cp -rvf /work/ui/index.cgi /rootfs/var/packages/pan-xunlei-com/target/xunlei-pan-cli-web && \
    mv /rootfs/var/packages/pan-xunlei-com/target/xunlei-pan-cli-launcher* /rootfs/var/packages/pan-xunlei-com/target/xunlei-pan-cli-launcher && \
    echo '{"port":5050, "internal": false, "dir":"/downloads"}' > /rootfs/var/packages/pan-xunlei-com/config.json && \
    mkdir -p /rootfs/usr/syno/synoman/webman/modules && \
    echo '#!/bin/bash' > /rootfs/usr/syno/synoman/webman/modules/authenticate.cgi && \
    echo 'echo dosk' >> /rootfs/usr/syno/synoman/webman/modules/authenticate.cgi && \
    chmod 755 /rootfs/usr/syno/synoman/webman/modules/authenticate.cgi && \
    mkdir -p /rootfs/downloads && mkdir -p /downloads && \
    rm -rvf /work
COPY --from=builder /go/src/xunlei_cgi/xunlei_cgi /rootfs/var/packages/pan-xunlei-com/xunlei_cgi
WORKDIR /
ADD init /init
EXPOSE 5050
VOLUME ["/downloads"]
ENTRYPOINT ["/init"]

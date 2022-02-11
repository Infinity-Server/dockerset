FROM golang:1.17.6 AS builder
WORKDIR /go/src
COPY xunlei_cgi /go/src/xunlei_cgi
WORKDIR /go/src/xunlei_cgi
RUN go env -w GO111MODULE=on && go env -w GOPROXY=https://goproxy.cn,direct
RUN CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -v
RUN ls -alth 

FROM ubuntu:20.04 AS rootfs
ARG VERSION=2.2.3
WORKDIR /work
RUN DEBIAN_FRONTEND=noninteractive apt-get -y update && \
    DEBIAN_FRONTEND=noninteractive apt-get -y install curl git xz-utils dpkg-dev
RUN mkdir -p /var/packages/pan-xunlei-com/shares && \
    mkdir -p /var/packages/pan-xunlei-com/target/var && \
    curl -L https://github.com/Infinity-Server/docker_image_set/raw/blob/xunlei_cgi/v${VERSION}-DSM7.x-$(dpkg-architecture -q DEB_BUILD_ARCH).spk --output /work/xunlei.spk && \
    tar -xvf xunlei.spk && \
    tar -xvf package.tgz && \
    cp -rvf /work/bin/bin/* /var/packages/pan-xunlei-com/target/ && \
    cp -rvf /work/ui/index.cgi /var/packages/pan-xunlei-com/target/xunlei-pan-cli-web && \
    mv /var/packages/pan-xunlei-com/target/xunlei-pan-cli-launcher* /var/packages/pan-xunlei-com/target/xunlei-pan-cli-launcher && \
    mkdir -p /usr/syno/synoman/webman/modules && \
    echo '#!/bin/bash' > /usr/syno/synoman/webman/modules/authenticate.cgi && \
    echo 'echo dosk' >> /usr/syno/synoman/webman/modules/authenticate.cgi && \
    chmod 755 /usr/syno/synoman/webman/modules/authenticate.cgi && \
    rm -rvf /work
COPY --from=builder /go/src/xunlei_cgi/xunlei_cgi /var/packages/pan-xunlei-com/xunlei_cgi

FROM alpine
COPY --from=rootfs /var/packages/pan-xunlei-com /root/pan-xunlei-com
COPY --from=rootfs /usr/syno/synoman/webman/modules/authenticate.cgi /usr/syno/synoman/webman/modules/authenticate.cgi
RUN apk add gcompat boost bash && \
    mkdir -p /downloads

WORKDIR /
ADD init /init
EXPOSE 5050
VOLUME ["/downloads"]
ENTRYPOINT ["/init"]

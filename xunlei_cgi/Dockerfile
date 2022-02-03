FROM golang:1.17.6 AS builder
WORKDIR /go/src
COPY src /go/src/xunlei_cgi
WORKDIR /go/src/xunlei_cgi
RUN go env -w GO111MODULE=on && go env -w GOPROXY=https://goproxy.cn,direct
RUN CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -v
RUN ls -alth 

FROM ubuntu:20.04
WORKDIR /work
RUN DEBIAN_FRONTEND=noninteractive apt-get -y update && \
    DEBIAN_FRONTEND=noninteractive apt-get -y install curl wget xz-utils dpkg-dev python3 python3-pip && \
    mkdir -p /var/packages/pan-xunlei-com/shares && \
    mkdir -p /var/packages/pan-xunlei-com/target
RUN curl -L https://github.com/Infinity-Server/docker_image_set/raw/blob/xunlei_cgi/v2.2.3-DSM7.x-$(dpkg-architecture -q DEB_BUILD_ARCH).spk --output /work/xunlei.spk && \
    tar -xvf xunlei.spk && \
    tar -xvf package.tgz && \
    cp -rvf /work/bin/bin/* /var/packages/pan-xunlei-com/target/ && \
    cp -rvf /work/ui/index.cgi /var/packages/pan-xunlei-com/target/xunlei-pan-cli-web && \
    mv /var/packages/pan-xunlei-com/target/xunlei-pan-cli-launcher* /var/packages/pan-xunlei-com/target/xunlei-pan-cli-launcher && \
    echo '{"port":5050, "internal": false, "dir":"/downloads"}' > /var/packages/pan-xunlei-com/config.json && \
    rm -rvf /work
COPY --from=builder /go/src/xunlei_cgi/xunlei_cgi /var/packages/pan-xunlei-com/xunlei_cgi
WORKDIR /var/packages/pan-xunlei-com
ENTRYPOINT ["/var/packages/pan-xunlei-com/xunlei_cgi"]
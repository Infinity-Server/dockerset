FROM ubuntu:20.04 AS builder

ARG VERSION=21.02.1
WORKDIR /work
COPY build_env /work/build_env
RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y curl dpkg-dev squashfs-tools && \
    mkdir /work/rootfs
RUN curl $(/work/build_env ${VERSION}) --output openwrt.img.gz
RUN gzip -d openwrt.img.gz && \
    fdisk -l openwrt.img && \
    unsquashfs -f -d /work/rootfs -o $(expr $(fdisk -l openwrt.img -o start | tail -n1 | xargs) \* 512) openwrt.img && \
    ls -alth /work/rootfs


FROM scratch
COPY --from=builder /work/rootfs /
ENTRYPOINT ["/sbin/init"]

FROM ubuntu:22.04 AS maker
ARG XUNLEI_VERSION=1.0.0.2
WORKDIR /install
COPY fake-deepin-elf-verify /install/fake-deepin-elf-verify
RUN DEBIAN_FRONTEND=noninteractive apt-get update -y && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y dpkg-dev curl && \
    dpkg-deb -b /install/fake-deepin-elf-verify /install/deepin.elf.verify.deb && \
    curl -fsSL "https://cdn-package-store6.deepin.com/appstore/pool/appstore/c/com.xunlei.download/com.xunlei.download_${XUNLEI_VERSION}_$(dpkg-architecture -q DEB_BUILD_ARCH).deb" -o /install/com.xunlei.download.deb

FROM ubuntu:22.04 AS rootfs
COPY --from=maker /install/deepin.elf.verify.deb /install/deepin.elf.verify.deb
COPY --from=maker /install/com.xunlei.download.deb /install/com.xunlei.download.deb
RUN DEBIAN_FRONTEND=noninteractive apt-get update -y && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl && \
    curl -fsSL https://xpra.org/xpra-2022.gpg -o /usr/share/keyrings/xpra-2022.gpg && \
    curl -fsSL https://xpra.org/repos/jammy/xpra.list -o /etc/apt/sources.list.d/xpra.list && \
    DEBIAN_FRONTEND=noninteractive apt-get update -y && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y libgtk2.0-0 libxss1 libnss3 libasound2 libdbus-glib-1-2 fonts-droid-fallback ttf-wqy-microhei xpra xpra-html5 && \
    apt install -y /install/deepin.elf.verify.deb /install/com.xunlei.download.deb 
COPY init /init

FROM scratch
ENV XPRA_PASSWORD=sksks
COPY --from=rootfs / /
EXPOSE 7010
ENTRYPOINT ["/init"]

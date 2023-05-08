FROM ubuntu:22.04 AS builder

ARG XUNLEI_VERSION=1.0.0.2
ENV DEBIAN_FRONTEND=noninteractive
WORKDIR /install

COPY fake-deepin-elf-verify /install/fake-deepin-elf-verify
RUN apt update -y && \
    apt install -y curl dpkg-dev && \
    dpkg-deb -b /install/fake-deepin-elf-verify /install/deepin.deb
RUN curl -fssl \
    "https://cdn-package-store6.deepin.com/appstore/pool/appstore/c/com.xunlei.download/com.xunlei.download_${XUNLEI_VERSION}_$(dpkg-architecture -q DEB_BUILD_ARCH).deb" \
    -o /install/xunlei.deb


FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive
ENV LC_ALL=en_US.UTF-8

RUN apt update -y && \
    apt install -y wget gnupg language-pack-zh-hans xfonts-100dpi \
    xfonts-75dpi xfonts-scalable xfonts-cyrillic fonts-ipafont-gothic \
    libgtk2.0-0 libxss1 libnss3 libasound2 libdbus-glib-1-2 fonts-droid-fallback \
    ttf-wqy-microhei && \
    locale-gen en_US.UTF-8 && \
    wget -q https://xpra.org/gpg.asc -O- | apt-key add - && \
    echo 'deb https://xpra.org/ jammy main' > /etc/apt/sources.list.d/xpra.list && \
    apt update -y && \
    apt install -y xpra && \
    find /usr/share/xpra/www -name "*.br" | xargs -I {} rm -rvf {} && \
    find /usr/share/xpra/www -name "*.gz" | xargs -I {} rm -rvf {}
COPY --from=builder /install/deepin.deb /tmp/deepin.deb
COPY --from=builder /install/xunlei.deb /tmp/xunlei.deb
RUN apt install -y /tmp/deepin.deb && apt install /tmp/xunlei.deb
RUN apt install -y pcmanfm && \
    cp /usr/share/applications/pcmanfm.desktop /tmp/pcmanfm.desktop && \
    rm -rvf /usr/share/applications/* && \
    cp /opt/apps/com.xunlei.download/entries/applications/com.xunlei.download.desktop /usr/share/applications/xunlei.desktop && \
    mv /tmp/pcmanfm.desktop /usr/share/applications/pcmanfm.desktop
ADD start /start
ADD rootfs /rootfs
RUN mv /usr/share/xpra/www/index.html /usr/share/xpra/www/remote.html && \
    cp -rvf /rootfs/* / && \
    rm -rvf /rootfs && \
    rm -rvf pcmanfm.desktop
CMD ["/start"]

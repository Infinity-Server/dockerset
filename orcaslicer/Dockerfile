# Build Xmin
FROM ubuntu:24.04 AS Xmin

# Install deps
RUN \
  apt-get update && \
  DEBIAN_FRONTEND=noninteractive \
  apt-get install --no-install-recommends -y \
    build-essential libx11-dev libgtk-3-dev

# Build Xmin
COPY Xmin.c /Xmin.c
RUN gcc -shared -fPIC -o /libXmin.so /Xmin.c -ldl -lX11

# Pull the baseimage
FROM jlesage/baseimage-gui:ubuntu-24.04-v4.10.6

# Install orcaslicer
RUN \
  echo "**** install packages ****" && \
  apt-get update && \
  DEBIAN_FRONTEND=noninteractive \
  apt-get install --no-install-recommends -y \
    curl \
    locales-all \
    ca-certificates \
    gstreamer1.0-alsa \
    gstreamer1.0-gl \
    gstreamer1.0-gtk3 \
    gstreamer1.0-libav \
    gstreamer1.0-plugins-bad \
    gstreamer1.0-plugins-base \
    gstreamer1.0-plugins-good \
    gstreamer1.0-plugins-ugly \
    gstreamer1.0-pulseaudio \
    gstreamer1.0-qt5 \
    gstreamer1.0-tools \
    gstreamer1.0-x \
    libgstreamer-plugins-bad1.0 \
    libwebkit2gtk-4.1-0 \
    fonts-wqy-microhei \
    libwx-perl && \
  echo "**** add icon ****" && \
  APP_ICON_URL="https://raw.githubusercontent.com/linuxserver/docker-templates/master/linuxserver.io/img/orcaslicer-logo.png" && \
  install_app_icon.sh "$APP_ICON_URL" && \
  echo "**** install orcaslicer from appimage ****" && \
  RELEASE_URL=$(curl -sX GET "https://api.github.com/repos/OrcaSlicer/OrcaSlicer/releases/latest"     | awk '/url/{print $4;exit}' FS='[""]') && \
  DOWNLOAD_URL=$(curl -sX GET "${RELEASE_URL}" | awk '/browser_download_url.*Ubuntu2404/{print $4;exit}' FS='[""]') && \
  cd /tmp && \
  curl -o \
    /tmp/orca.app -L \
    "${DOWNLOAD_URL}" && \
  chmod +x /tmp/orca.app && \
  ./orca.app --appimage-extract && \
  mv squashfs-root /opt/orcaslicer && \
  echo "**** cleanup ****" && \
  mkdir -p /config/xdg/config && \
  apt-get autoclean && \
  rm -rf \
    /config/.cache \
    /config/.launchpadlib \
    /var/lib/apt/lists/* \
    /var/tmp/* \
    /tmp/*

# App run stuff
ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt
COPY startapp.sh /startapp.sh
COPY --from=Xmin /libXmin.so /libXmin.so

# Set the application name
RUN set-cont-env APP_NAME "OrcaSlicer"

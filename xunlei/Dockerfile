# Base image
FROM ubuntu:20.04

# Build args
ARG XUNLEI_VERSION=1.0.0.2

# Init installation
ENV XPRA_PASSWORD=sksks
ENV LC_ALL=en_US.UTF-8
ENV PUID=1001
ENV PGID=100
RUN mkdir /install
WORKDIR /install

# Install
## Base environment
RUN apt-get update
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends tzdata keyboard-configuration
RUN DEBIAN_FRONTEND=noninteractive apt-get -y install \
  wget gnupg \
  language-pack-zh-hans \
  make gcc \
  xfonts-100dpi \
  xfonts-75dpi \
  xfonts-scalable \
  xfonts-cyrillic \
  fonts-ipafont-gothic
RUN locale-gen en_US.UTF-8
RUN wget -q https://xpra.org/gpg.asc -O- | apt-key add -
RUN echo 'deb https://xpra.org/ focal main' > /etc/apt/sources.list.d/xpra.list
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y pcmanfm xpra
RUN useradd -ms /bin/bash -u $PUID -g $PGID dosk
RUN mkdir -p /etc/xdg/menus
COPY hook /usr/share/xpra/www/hook
COPY menu /etc/xdg/menus/kde-debian-menu.menu
RUN rm -rf /usr/share/applications/pcmanfm-desktop-pref.desktop /usr/share/applications/debian-uxterm.desktop
RUN rm /usr/share/xpra/www/index.html.* && sed -i "s/<title>xpra websockets client<\/title>/<title>SpringHack<\/title><script src='hook\/hook.js'><\/script><link rel='stylesheet' href='hook\/hook.css'>/" /usr/share/xpra/www/index.html
COPY init /init
COPY start /start

# Install
## UI Components
COPY fake-deepin-elf-verify /install/fake-deepin-elf-verify
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y dpkg-dev curl libgtk2.0-0 libxss1 libnss3 libasound2 libdbus-glib-1-2 fonts-droid-fallback ttf-wqy-microhei gosu
RUN dpkg-deb -b /install/fake-deepin-elf-verify /install/deepin.elf.verify.deb
RUN apt install -y /install/deepin.elf.verify.deb
RUN curl -fsSL "https://cdn-package-store6.deepin.com/appstore/pool/appstore/c/com.xunlei.download/com.xunlei.download_${XUNLEI_VERSION}_$(dpkg-architecture -q DEB_BUILD_ARCH).deb" -o /install/com.xunlei.download.deb
RUN apt install -y /install/com.xunlei.download.deb

# Install
# Cleanup
WORKDIR /home/dosk
RUN rm -rvf /install
RUN apt-get autoremove -y && \
apt-get clean && \
apt-get autoclean

# Arguments
EXPOSE 7010
VOLUME ["/home/dosk/downloads"]

# Start
CMD /init

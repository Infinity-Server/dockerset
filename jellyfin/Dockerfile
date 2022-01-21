ARG JELLYFIN_VERSION=10.8.0-alpha5
FROM jellyfin/jellyfin:${JELLYFIN_VERSION}

ARG DRIVER_URL="https://github.com/Infinity-Server/docker_image_set/raw/blob/jellyfin/intel-media-va-driver-non-free_21.1.1%2Bds1-1_amd64.deb"
ARG FFMPEG_URL="https://github.com/Infinity-Server/docker_image_set/raw/blob/jellyfin/jellyfin-ffmpeg_4.4.1-1-bullseye_amd64.deb"

ENV PUID=1001
ENV PGID=100
RUN useradd -ms /bin/bash -u $PUID -g $PGID dosk

RUN apt update && apt install gcc curl gosu unzip dpkg-dev -y

COPY post_setup /post_setup
COPY ffmpeg.c /usr/lib/jellyfin-ffmpeg/ffmpeg.c
RUN /post_setup "${DRIVER_URL}" "${FFMPEG_URL}"
RUN apt clean

COPY init /init
ENTRYPOINT ["/init"]

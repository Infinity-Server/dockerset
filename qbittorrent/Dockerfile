FROM caddy AS builder
ARG VUE_TORRENT_VERSION=2.19.0
ADD build_env /build_env
WORKDIR /
RUN apk add jq curl bash unzip
RUN /build_env ${VUE_TORRENT_VERSION}


FROM linuxserver/qbittorrent
ENV WEB_PORT=18080
COPY --from=builder /usr/bin/caddy /caddy
COPY --from=builder /vuetorrent/public /vuetorrent
ADD root/ /

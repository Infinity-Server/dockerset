ARG ND_VER=latest

#####################################################
### build taglib
FROM public.ecr.aws/docker/library/alpine:3.21 AS taglib
ARG TARGETPLATFORM
ARG CROSS_TAGLIB_VERSION=2.0.2-1
ENV CROSS_TAGLIB_RELEASES_URL=https://github.com/navidrome/cross-taglib/releases/download/v${CROSS_TAGLIB_VERSION}/
RUN <<EOT
    PLATFORM=$(echo ${TARGETPLATFORM} | tr '/' '-')
    FILE=taglib-${PLATFORM}.tar.gz

    DOWNLOAD_URL=${CROSS_TAGLIB_RELEASES_URL}${FILE}
    wget ${DOWNLOAD_URL}

    mkdir /taglib
    tar -xzf ${FILE} -C /taglib
EOT

#####################################################
### build navidrome
FROM public.ecr.aws/docker/library/golang:1.23-bookworm AS builder
RUN apt update -y && apt install -y binutils gcc g++ libc6-dev zlib1g-dev
ARG ND_VER=latest
WORKDIR /go/src
COPY . /go/src/main
RUN --mount=from=taglib,target=/taglib,src=/taglib,ro <<EOT
    git clone https://github.com/navidrome/navidrome /go/src/navidrome
    cd /go/src/navidrome
    cp -rvf /go/src/main/main.go ./main.go
    export CGO_ENABLED=1
    export PKG_CONFIG_PATH=/taglib/lib/pkgconfig
    go mod tidy
    go build -a -v -tags=netgo -gcflags="all=-l" -ldflags="-extldflags '-static -latomic' -w -s \
      -X github.com/navidrome/navidrome/consts.gitSha=SpringHack \
      -X github.com/navidrome/navidrome/consts.gitTag=$ND_VER"
EOT

#####################################################
### build gorip
FROM golang:1.23.1-bookworm AS gorip
WORKDIR /code
RUN apt update -y && apt install -y git
RUN git clone https://github.com/woesbot/gorip && cd gorip && go build

#####################################################
### extract frontend assets
FROM deluan/navidrome:$ND_VER AS assets
WORKDIR /code
COPY --from=gorip /code/gorip/gorip /bin/gorip
RUN gorip -e /app/navidrome && ls -alth /code

#####################################################
### build final image
FROM alpine:3.20
LABEL maintainer="springhack@lvie.cn"
RUN apk add -U --no-cache ffmpeg mpv libtag
RUN ffmpeg -buildconf
COPY --from=builder /go/src/navidrome/navidrome /app/navidrome
COPY --from=assets /code/build /app/build
VOLUME ["/data", "/music"]
ENV ND_MUSICFOLDER=/music
ENV ND_DATAFOLDER=/data
ENV ND_PORT=4533
EXPOSE ${ND_PORT}
WORKDIR /app
ENTRYPOINT ["/app/navidrome"]

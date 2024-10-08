ARG ND_VER=latest

#####################################################
### build navidrome
FROM deluan/ci-goreleaser AS builder
ARG ND_VER=latest
WORKDIR /go/src
COPY . /go/src/navidrome
RUN cd /go/src/navidrome && \
    go get github.com/navidrome/navidrome@v$ND_VER && \
    go mod tidy && \
    go build \
        -a -v \
        -gcflags="all=-l" \
        -ldflags="-extldflags -static -X github.com/navidrome/navidrome/consts.gitSha=SpringHack -X github.com/navidrome/navidrome/consts.gitTag=$ND_VER"

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

ARG ND_VER=latest

#####################################################
### build navidrome
FROM golang:1.23.1-bookworm AS builder
ARG ND_VER=latest
WORKDIR /go/src
COPY . /go/src/navidrome
RUN apt update -y && apt install -y build-essential libtag1-dev pkg-config
RUN cd /go/src/navidrome && \
    go get github.com/navidrome/navidrome@v$ND_VER && \
    go mod tidy && \
    go build \
        -a -v \
        -gcflags="all=-l" \
        -ldflags="-X github.com/navidrome/navidrome/consts.gitSha=SpringHack -X github.com/navidrome/navidrome/consts.gitTag=$ND_VER"

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
FROM debian:bookworm
LABEL maintainer="springhack@lvie.cn"
RUN apt update -y && apt install -y ffmpeg mpv libtag1-dev && apt clean
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

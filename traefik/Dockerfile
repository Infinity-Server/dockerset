ARG TK_VER=latest

#####################################################
### build gorip
FROM golang:1.23.1-bookworm AS gorip
WORKDIR /code
RUN apt update -y && apt install -y git
RUN git clone https://github.com/woesbot/gorip && cd gorip && go build

#####################################################
### extract frontend assets
FROM traefik:v$TK_VER AS assets
WORKDIR /code
COPY --from=gorip /code/gorip/gorip /bin/gorip
RUN gorip -e $(which traefik) && ls -lath /code/static/

#####################################################
### build traefik
FROM golang:1.24.0-bookworm AS traefik
ARG TK_VER=latest
WORKDIR /code
RUN git clone -b v$TK_VER https://github.com/traefik/traefik
COPY --from=assets /code/static traefik/webui/static
RUN ls -alth traefik/webui/static
ADD hook.go traefik/cmd/traefik/hook.go
RUN cd traefik && go mod tidy && CGO_ENABLED=0 GOGC=off go build -ldflags "-s -w -X github.com/traefik/traefik/v3/pkg/version.Version=v$TK_VER -X github.com/traefik/traefik/v3/pkg/version.Codename=nginx -X github.com/traefik/traefik/v3/pkg/version.BuildDate=1970.1.1" ./cmd/traefik

#####################################################
### build final image
FROM alpine:3.21
RUN apk add --no-cache --no-progress ca-certificates tzdata
COPY --from=traefik /code/traefik/traefik /traefik
EXPOSE 80
VOLUME ["/tmp"]
ENTRYPOINT ["/traefik"]

FROM golang AS builder
WORKDIR /go/src
COPY caddy /go/src/caddy
WORKDIR /go/src/caddy
RUN go env -w GO111MODULE=on && go env -w GOPROXY=https://goproxy.cn,direct
RUN cd cmd/caddy && CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -v

FROM alpine
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk --no-cache add ca-certificates
COPY --from=builder /go/src/caddy/cmd/caddy/caddy /caddy
ENTRYPOINT ["/caddy"]

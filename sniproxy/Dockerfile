FROM golang:1.23 AS builder
WORKDIR /go/src
COPY src /go/src/sniproxy
WORKDIR /go/src/sniproxy
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static" -X github.com/ameshkov/sniproxy/internal/version.VersionString=dockerset' -v

FROM alpine:latest
RUN apk --no-cache add ca-certificates
ENV SNI_SOCKS_PROXY="socks5://127.0.0.1:1080"
COPY --from=builder /go/src/sniproxy/sniproxy /sniproxy
CMD exec /sniproxy --dns-redirect-ipv4-to=127.0.0.1 --dns-redirect-ipv6-to=::1 --forward-proxy=$SNI_SOCKS_PROXY

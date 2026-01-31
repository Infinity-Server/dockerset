FROM golang:1.24.0 AS builder
WORKDIR /go/src
COPY . /go/src/creality-discovery
WORKDIR /go/src/creality-discovery
RUN go env -w GO111MODULE=on && go env -w GOPROXY=https://goproxy.cn,direct
RUN CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -v

FROM alpine:3.16.2
RUN apk --no-cache add ca-certificates
COPY --from=builder /go/src/creality-discovery/creality-discovery /creality-discovery
ENTRYPOINT ["/creality-discovery"]

FROM golang:1.17.6 AS builder
WORKDIR /go/src
COPY dash/app /go/src/dash
RUN cd /go/src/dash/server && CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -v index.go

FROM alpine:3.16.2
RUN apk --no-cache add bash procps ca-certificates
COPY dash/app /dash
COPY --from=builder /go/src/dash/server/index /dash/server/index
WORKDIR /dash/server
ENTRYPOINT ["/dash/server/index"]

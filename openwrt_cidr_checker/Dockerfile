FROM golang as builder
COPY checker /go/src/checker
WORKDIR /go/src/checker
RUN CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -v

FROM alpine
COPY --from=builder /go/src/checker/checker /checker
ENV OPENWRT_HOST=http://10.10.10.1
ENV OPENWRT_USER=root
ENV OPENWRT_PASS=
ENV OPENWRT_IFACE=lan
ENV LISTEN_PORT=3000
ENV STATUS_OK=200
ENV STATUS_FAIL=444
ENTRYPOINT ["/checker"]

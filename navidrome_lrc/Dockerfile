FROM golang as builder
COPY lrc /go/src/lrc
WORKDIR /go/src/lrc
RUN CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -v

FROM alpine
COPY --from=builder /go/src/lrc/lrc /lrc
ENV LISTEN_PORT="4555"
ENV FORWARD_SERVER="127.0.0.1:4533"
ENTRYPOINT ["/lrc"]

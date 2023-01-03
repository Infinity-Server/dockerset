FROM golang as builder
COPY rewriter /go/src/rewriter
WORKDIR /go/src/rewriter
RUN CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -v

FROM alpine
COPY --from=builder /go/src/rewriter/rewriter /rewriter
ENV URL="/"
ENV STAGE="request"
ENV METHOD="GET"
ENV SEARCH="title"
ENV REPLACEMENT="balabala"
ENV FORWARD_SERVER="127.0.0.1:8080"
ENV LISTEN_PORT="3000"
ENTRYPOINT ["/rewriter"]

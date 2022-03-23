FROM golang:1.17.6 AS builder
WORKDIR /go/src/github.com/google/huproxy
COPY huproxy .
RUN mkdir /app
RUN go env -w GO111MODULE=on && go env -w GOPROXY=https://goproxy.cn,direct
RUN CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o /app .
RUN CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o /app ./huproxyclient

FROM alpine
WORKDIR /
COPY --from=0 /app/ /bin
CMD ["/bin/huproxy"]

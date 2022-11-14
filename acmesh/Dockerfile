FROM golang:alpine AS builder
RUN apk add --no-cache git curl ca-certificates socat bash openssl
WORKDIR /workspace
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -o webhook -ldflags '-w -extldflags "-static"' .
RUN curl -fsSL https://get.acme.sh | sh -s email=infinity-server@dosk.host

FROM alpine
COPY --from=builder /workspace/webhook /usr/local/bin/webhook
COPY --from=builder /root/.acme.sh /root/.acme.sh
ADD acme_delegate /root/acme_delegate
RUN apk add --no-cache ca-certificates curl socat bash openssl && chmod 755 /root/acme_delegate

ENTRYPOINT ["webhook"]

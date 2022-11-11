FROM golang:1.16-alpine AS build_deps
RUN apk add --no-cache git curl ca-certificates socat bash openssl
WORKDIR /workspace
COPY go.mod .
COPY go.sum .
RUN go mod download

FROM build_deps AS build
COPY . .
RUN CGO_ENABLED=0 go build -o webhook -ldflags '-w -extldflags "-static"' .
RUN curl -fsSL https://get.acme.sh | sh -s email=infinity-server@dosk.host

FROM alpine
COPY --from=build /workspace/webhook /usr/local/bin/webhook
COPY --from=build /root/.acme.sh /root/.acme.sh
ADD acme_delegate /root/acme_delegate
RUN apk add --no-cache ca-certificates curl socat bash openssl && chmod 755 /root/acme_delegate

ENTRYPOINT ["webhook"]

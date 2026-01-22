FROM golang:1.25-alpine AS builder
RUN apk add --no-cache ca-certificates git curl
WORKDIR /app
COPY . ./
RUN mkdir -p /usr/local/share/xray && curl -L -o /usr/local/share/xray/geosite.dat https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat
RUN go build -o /usr/local/bin/geo-sni-proxy .

FROM alpine
COPY --from=builder /usr/local/share/xray/geosite.dat /usr/local/share/xray/geosite.dat
COPY --from=builder /usr/local/bin/geo-sni-proxy /usr/local/bin/geo-sni-proxy
ENTRYPOINT ["/usr/local/bin/geo-sni-proxy"]

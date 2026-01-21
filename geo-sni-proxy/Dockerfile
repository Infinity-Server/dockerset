FROM golang:1.25-alpine

# Install basic build dependencies
RUN apk add --no-cache ca-certificates git curl

WORKDIR /app

COPY . ./

# Pre-fetch domain-list-community dat for geosite rules
RUN mkdir -p /usr/local/share/xray && curl -L -o /usr/local/share/xray/geosite.dat https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat

# Build static binary
RUN go build -o /usr/local/bin/geo-sni-proxy .

EXPOSE 53/udp 53/tcp 80/tcp 443/tcp

ENTRYPOINT ["/usr/local/bin/geo-sni-proxy"]

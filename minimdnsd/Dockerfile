FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY cmd/minimdnsd/main.go ./cmd/minimdnsd/main.go
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /minimdnsd ./cmd/minimdnsd

FROM alpine
COPY --from=builder /minimdnsd /minimdnsd
ENTRYPOINT ["/minimdnsd"]

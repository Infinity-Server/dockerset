# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build
ARG TARGETOS=linux
ARG TARGETARCH=amd64
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -o /out/netfin ./cmd/sidecar

FROM scratch
COPY --from=build /out/netfin /netfin
ENTRYPOINT ["/netfin"]

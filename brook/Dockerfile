FROM alpine

ARG BROOK_VERSION=v20220707

RUN apk add --no-cache curl && \
    curl -fsSL https://github.com/txthinking/brook/releases/download/${BROOK_VERSION}/brook_linux_amd64 -o /bin/brook && \
    chmod 755 /bin/brook

ENTRYPOINT ["/bin/brook"]

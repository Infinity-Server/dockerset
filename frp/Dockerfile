FROM ubuntu:20.04 AS builder

WORKDIR /
ARG VERSION=0.38.0
RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y jq curl dpkg-dev
ADD build_env /build_env
RUN /build_env ${VERSION}


FROM alpine:3.16.2

COPY --from=builder /frp /frp
RUN apk --no-cache add ca-certificates && \
    ln -s /frp/frps /bin/frps && \
    ln -s /frp/frpc /bin/frpc

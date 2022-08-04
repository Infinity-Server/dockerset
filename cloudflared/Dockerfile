FROM dockage/alpine:3.16-openrc

RUN apk --no-cache add curl gcompat && \
    curl -fsSL https://github.com/cloudflare/cloudflared/releases/download/2022.7.1/cloudflared-linux-amd64 -o /bin/cloudflared && \
    chmod 755 /bin/cloudflared

ADD cf /etc/init.d/cf

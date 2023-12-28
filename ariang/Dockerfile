FROM alpine AS downloader
ARG VERSION=1.3.7
RUN apk add unzip curl
RUN curl -fsSL -vvv https://github.com/mayswind/AriaNg/releases/download/${VERSION}/AriaNg-${VERSION}-AllInOne.zip -o /aio.zip
RUN unzip -j /aio.zip index.html -d /

FROM busybox
WORKDIR /www
COPY --from=downloader /index.html /www/index.html
COPY --from=downloader /index.html /www/ariang/index.html
CMD ["busybox", "httpd", "-f", "-v", "-p", "3000"]

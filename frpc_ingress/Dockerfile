FROM springhack/frp AS downloader

FROM flant/shell-operator:latest
COPY --from=downloader /frp /frp
RUN apk --no-cache add nodejs screen && mkdir /frp/client
ADD utils /utils
ADD hooks.sh /hooks/hooks.sh

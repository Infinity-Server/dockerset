FROM springhack/frp AS downloader

FROM flant/shell-operator:latest
COPY --from=downloader /frp /frp
RUN apk --no-cache add nodejs screen
ADD app /app
ADD frpc-hooks.sh /hooks/frpc-hooks.sh

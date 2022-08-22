FROM alpine:3.16.2

ENV API_TOKEN='ksfbdjanujfbkbajbsfgkbcal'
ENV API_TARGET_DOMAIN_4='test.dosk.win'
ENV API_ROOT_DOMAIN='dosk.win'
ENV TIMER_INTERVAL=300
ENV INTERFACE='eth0'

RUN apk add jq curl bash
ADD timer /timer
ADD cloudflare_ddns /cloudflare_ddns

ENTRYPOINT ["/timer"]

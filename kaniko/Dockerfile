FROM bitnami/kaniko:latest AS assets

FROM alpine:latest
COPY --from=assets /kaniko /kaniko
COPY --from=assets /workspace /workspace
COPY --from=assets /etc/nsswitch.conf /etc/nsswitch.conf

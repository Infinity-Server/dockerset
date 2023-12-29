FROM bitnami/kubectl:latest AS kubectl

FROM caddy:latest
ENV API_VERSION=v1alpha1
ENV API_GROUP=traefik.io
ENV API_CRD=ingressroutes
COPY --from=kubectl /opt/bitnami/kubectl/bin/kubectl /bin/kubectl
ADD assets /etc/caddy
ADD init /init
CMD ["/init"]

ARG ZT_VERSION=latest
FROM zerotier/zerotier:${ZT_VERSION}

RUN DEBIAN_FRONTEND=noninteractive apt-get update -y && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y kmod
COPY init /init

EXPOSE 9993/udp
ENTRYPOINT ["/init"]
CMD ["-U"]

FROM alpine

WORKDIR /work
RUN apk add bash docker && \
    wget https://raw.githubusercontent.com/eficode/wait-for/master/wait-for && \
    chmod 755 /work/wait-for
ADD init /work/init

VOLUME ["/var/run/docker.sock"]
ENTRYPOINT ["/work/init"]

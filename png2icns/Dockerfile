FROM ubuntu:latest

RUN DEBIAN_FRONTEND=noninteractive apt-get update -y && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y icnsutils

ENTRYPOINT ["png2icns"]

FROM ubuntu:20.04

WORKDIR /
COPY init /init

RUN apt-get update && \
  apt-get upgrade -y && \
  apt-get install zsh language-pack-zh-hans -y && \
  locale-gen zh_CN.UTF-8

ENTRYPOINT ["/bin/sh", "-c", "/init"]

FROM ubuntu

RUN apt update && \
  apt install -y python3-pip && \
  pip3 install https://github.com/shadowsocks/shadowsocks/archive/master.zip && \
  rm -rf /var/lib/apt/lists/*

ENV SERVER_ADDR=0.0.0.0
ENV SERVER_PORT=7999
ENV PASSWORD=sksks
ENV METHOD=aes-256-cfb
ENV TIMEOUT=300
ENV FASTOPEN=false

ADD init /init
ENTRYPOINT ["/init"]

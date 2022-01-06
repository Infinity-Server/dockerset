FROM jellyfin/jellyfin:latest

ENV PUID=1001
ENV PGID=100

COPY init /init
RUN apt update && apt install gosu -y && apt clean

ENTRYPOINT ["/init"]

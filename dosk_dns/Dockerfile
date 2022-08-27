FROM node:alpine3.16

WORKDIR /
RUN apk --no-cache add git && \
    git clone https://github.com/Infinity-Server/dosk_dns.git && \
    cd dosk_dns && \
    npm i
WORKDIR /dosk_dns

ENTRYPOINT ["node", "dns.js"]

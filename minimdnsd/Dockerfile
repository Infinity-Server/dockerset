FROM ubuntu AS builder
ADD minimdnsd.c /minimdnsd.c
RUN apt update -y && apt install -y build-essential && gcc /minimdnsd.c -o /minimdnsd -static -Wall -pedantic -Os -g -flto -ffunction-sections -Wl,--gc-sections -fdata-sections

FROM alpine
COPY --from=builder /minimdnsd /minimdnsd
ENTRYPOINT ["/minimdnsd"]

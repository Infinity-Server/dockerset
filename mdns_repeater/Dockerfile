FROM alpine AS builder
ARG HGVERSION=springhack
ADD mdns-repeater.c mdns-repeater.c
RUN apk add build-base && \
    gcc -o /bin/mdns-repeater mdns-repeater.c -DHGVERSION=\"${HGVERSION}\"

FROM alpine
COPY --from=builder /bin/mdns-repeater /bin/mdns-repeater
ENTRYPOINT ["/bin/mdns-repeater"]

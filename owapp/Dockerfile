FROM ubuntu:20.04

ARG URL="https://cdn.ionewu.com/upgrade/d/DSM6_owapp-x86_64-2.3.21-6119.spk"

WORKDIR /work
RUN DEBIAN_FRONTEND=noninteractive apt-get -y update && \
    DEBIAN_FRONTEND=noninteractive apt-get -y install curl wget xz-utils
RUN mkdir -p /volume1/OWDownload && \
    curl ${URL} --output owapp.spk && \
    tar -xvf owapp.spk && \
    tar -xvf package.tgz && \
    mv owapp /root/owapp && \
    sed -i 's/DID=/DID="A8ODN02468" #/g' /root/owapp/bin/owapp_monitor && \
    sed -i 's/DN=/DN="DS3617xs-j" #/g' /root/owapp/bin/owapp_monitor

WORKDIR /root
EXPOSE 8283
VOLUME ["/volume1/OWDownload"]
ENTRYPOINT ["sh", "/root/owapp/bin/owapp_monitor"]

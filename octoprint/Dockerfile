FROM octoprint/octoprint:latest

RUN apt update && \
    apt install -y jq git curl dpkg-dev && \
    git clone https://github.com/hellerbarde/OctoPrint-Network-Printing && \
    cd OctoPrint-Network-Printing && /usr/local/bin/python setup.py install && cd .. && rm -rvf OctoPrint-Network-Printing

ARG FRP_VERSION=0.38.0
ADD run /etc/services.d/frp/run
COPY post_setup .
RUN ./post_setup ${FRP_VERSION}

EXPOSE 7000

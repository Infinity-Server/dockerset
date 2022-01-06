FROM ubuntu:20.04

# Init installation
ENV USER=root XAUTHORITY=/xauthority/.docker.xauth DISPLAY=":0"
ENV LC_ALL=en_US.UTF-8
ENV GEOMETRY 1770x920
ENV PASSWORD=sksks
RUN mkdir /xauthority
RUN mkdir /install
WORKDIR /install

# Install
## Base environment
RUN DEBIAN_FRONTEND=noninteractive apt-get update && apt-get install -y --no-install-recommends tzdata
RUN DEBIAN_FRONTEND=noninteractive apt-get -y install \
language-pack-zh-hans \
tightvncserver \
expect \
make gcc \
xfonts-100dpi \
xfonts-75dpi \
xfonts-scalable \
xfonts-cyrillic \
tightvncserver \
fonts-ipafont-gothic && \
expect -c 'set timeout 3;spawn /usr/bin/vncpasswd;expect "*?assword:*";send -- "selenium\r";expect "*?erify:*";send -- "selenium\r";expect "*?view-only password*";send -- "n\r";send -- "\r";expect eof' && \
touch /root/.xsession && \
apt-get remove --purge -y expect && \
apt-get autoremove -y && \
apt-get clean && \
apt-get autoclean
COPY init /init
COPY vncpwd /install/vncpwd
RUN cd vncpwd && make && ./vncpwd ~/.vnc/passwd set ${PASSWORD}

# Install
## UI Components
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y firefox fonts-droid-fallback ttf-wqy-microhei i3-wm autocutsel novnc websockify && \
cp -rvf /usr/share/novnc/vnc_lite.html /usr/share/novnc/index.html && \
apt-get autoremove -y && \
apt-get clean && \
apt-get autoclean

# Start
EXPOSE 5070
CMD /init

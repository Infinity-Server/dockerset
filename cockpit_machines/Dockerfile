FROM ubuntu:20.04

# using systemd as init system, and need run with --privileged --interactive --tty and --device /dev/kvm:/dev/kvm
ENV container=docker

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y cockpit cockpit-machines \
      qemu qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils virt-manager && \
    rm /lib/systemd/system/NetworkManager-wait-online.service && \
    rm /lib/systemd/system/NetworkManager.service
RUN yes sksks | passwd
COPY reverse_proxy /etc/init.d/reverse_proxy
RUN /lib/systemd/systemd-sysv-install enable reverse_proxy

ENV COCKPIT_ORIGINS=http://127.0.0.1:9090
EXPOSE 9090
ENTRYPOINT ["/usr/sbin/init"]

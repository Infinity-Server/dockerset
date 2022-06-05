FROM docker:dind AS rootfs
RUN export | grep "DOCKER\|DIND" > /etc/dind.env
RUN sed -i 's/#!\/bin\/sh/#!\/bin\/sh\n\nsource \/etc\/dind.env/g' /usr/local/bin/dockerd-entrypoint.sh

FROM scratch
COPY --from=rootfs / /
EXPOSE 2375 2376
ENTRYPOINT ["dockerd-entrypoint.sh"]
CMD []

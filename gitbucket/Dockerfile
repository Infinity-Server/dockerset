FROM gitbucket/gitbucket

ARG DOCKER_COMPOSE_VERSION=v2.6.0

RUN curl -fsSL https://get.docker.io | bash -E -
RUN curl -fsSL https://github.com/docker/compose/releases/download/$DOCKER_COMPOSE_VERSION/docker-compose-linux-x86_64 -o /usr/bin/docker-compose
RUN chmod 755 /usr/bin/docker-compose

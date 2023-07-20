FROM php

ADD src /code
RUN mkdir /data

ENV WEBAUTHN_RP_ID=localhost
ENV WEBAUTHN_RP_NAME=LOCALHOST
ENV WEBAUTHN_SUCCESS_INCLUDE=none

EXPOSE 8000
WORKDIR /code/Public

ENTRYPOINT ["php"]
CMD ["-S", "0.0.0.0:8000", "webauthn.php"]

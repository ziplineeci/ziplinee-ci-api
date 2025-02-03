FROM scratch

ENV GRACEFUL_SHUTDOWN_DELAY_SECONDS="20" \
    ZIPLINEE_LOG_FORMAT="json" \
    ZIPLINEE_GIT_NAME="ziplinee-ci-api"

LABEL maintainer="ziplinee.io" \
      description="The ${ZIPLINEE_GIT_NAME} is the component that handles api requests and starts build jobs using the ziplinee-ci-builder"

#COPY ca-certificates.crt /etc/ssl/certs/
COPY publish/${ZIPLINEE_GIT_NAME} /

ENTRYPOINT ["/${ZIPLINEE_GIT_NAME}"]

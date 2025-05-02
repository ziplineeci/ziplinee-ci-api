FROM scratch

# You can keep runtime env vars here
ENV GRACEFUL_SHUTDOWN_DELAY_SECONDS="20" \
    ZIPLINEE_LOG_FORMAT="json" \
    ZIPLINEE_GIT_NAME="ziplinee-ci-api"

LABEL maintainer="ziplinee.io" \
      description="The ziplinee-ci-api is the component that handles API requests and starts build jobs using the ziplinee-ci-builder"

COPY publish/ziplinee-ci-api /ziplinee-ci-api

ENTRYPOINT ["/ziplinee-ci-api"]
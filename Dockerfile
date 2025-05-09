# Stage 1: Install certs and prepare binary
FROM alpine:3.18 AS base
RUN apk add --no-cache ca-certificates
COPY publish/ziplinee-ci-api /ziplinee-ci-api

# Stage 2: Build secure scratch image
FROM scratch

# Copy certs to default trust location
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy binary
COPY --from=base /ziplinee-ci-api /ziplinee-ci-api

# Env vars
ENV GRACEFUL_SHUTDOWN_DELAY_SECONDS="20" \
    ZIPLINEE_LOG_FORMAT="json" \
    ZIPLINEE_GIT_NAME="ziplinee-ci-api"

ENTRYPOINT ["/ziplinee-ci-api"]
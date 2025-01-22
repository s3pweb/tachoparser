# syntax=docker/dockerfile:1

# Python stage to download certificates
FROM python:3.10-slim-buster AS pythonbuilder
ENV PYTHONUNBUFFERED=1
RUN pip install requests
RUN pip install lxml
RUN mkdir /scripts
RUN mkdir /internal
COPY ./scripts/ /scripts/
COPY ./internal/ /internal/
WORKDIR /scripts/pks1
RUN ./dl_all_pks1.py
WORKDIR /scripts/pks2
RUN ./dl_all_pks2.py

# Go builder stage
FROM golang:1.19 AS gobuilder
WORKDIR /go/src/github.com/s3pweb/tachoparser
COPY ./ ./
COPY --from=pythonbuilder /internal/pkg/certificates/pks1/ internal/pkg/certificates/pks1/
COPY --from=pythonbuilder /internal/pkg/certificates/pks2/ internal/pkg/certificates/pks2/
RUN go mod vendor
WORKDIR /go/src/github.com/s3pweb/tachoparser/cmd/dddhttp
RUN go build -o dddhttp .

# Final stage with Ubuntu
FROM ubuntu:22.04

# Install necessary certificates and timezone data
RUN apt-get update && apt-get install -y \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -m -s /bin/bash appuser

# Copy binary and certificates
COPY --from=gobuilder /go/src/github.com/s3pweb/tachoparser/cmd/dddhttp/dddhttp /usr/local/bin/
COPY --from=pythonbuilder /internal/pkg/certificates/pks1/ /internal/pkg/certificates/pks1/
COPY --from=pythonbuilder /internal/pkg/certificates/pks2/ /internal/pkg/certificates/pks2/

# Set ownership for the certificates
RUN chown -R appuser:appuser /internal

# Switch to non-root user
USER appuser

# Expose the HTTP port
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/dddhttp"]
CMD []
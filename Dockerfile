# ---- Stage 0 ----
# Builds media repo binaries
FROM golang:1.20-alpine AS builder

WORKDIR /opt/collector
COPY . /opt
RUN go build -o /opt/bin/collector /opt/collector/main.go

# ---- Stage 1 ----
# Final runtime stage.
FROM alpine

ENV PGOF_BIND_ADDRESS=":8080"
ENV PGOF_DIRECTORY="/data"
ENV PGOF_SUBMIT_AUTH_KEYS_FILE="/secret/submit_keys"
ENV PGOF_MERGE_AUTH_KEYS_FILE="/secret/merge_keys"
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=1m --timeout=5s CMD curl -f http://localhost:8080/healthz || exit 1

COPY --from=builder /opt/bin/collector /usr/local/bin/collector

CMD /usr/local/bin/collector

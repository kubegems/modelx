# syntax=docker/dockerfile:1
FROM alpine
# TARGETOS TARGETARCH already set by '--platform'
ARG TARGETOS TARGETARCH 
COPY modelxd-${TARGETOS}-${TARGETARCH} /app/modelxd
WORKDIR /app
ENTRYPOINT ["/app/modelxd"]

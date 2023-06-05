# syntax=docker/dockerfile:1
FROM alpine
# TARGETOS TARGETARCH already set by '--platform'
ARG TARGETOS TARGETARCH
COPY modelxd-${TARGETOS}-${TARGETARCH} /bin/modelxd
COPY modelx-${TARGETOS}-${TARGETARCH} /bin/modelx
WORKDIR /app
ENTRYPOINT ["/bin/modelxd"]

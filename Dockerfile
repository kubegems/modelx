FROM alpine
ENV OS=linux ARCH=amd64
COPY bin/modelxd-${OS}-${ARCH} /app/
WORKDIR /app
ENTRYPOINT ["/app/modelxd-${OS}-${ARCH}"]

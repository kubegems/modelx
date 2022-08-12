FROM alpine
ENV OS=linux ARCH=amd64
COPY bin/modelxd-linux-amd64 /app/
WORKDIR /app
ENTRYPOINT ["/app/modelxd-linux-amd64"]

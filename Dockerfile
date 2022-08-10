FROM alpine
COPY bin/modelx bin/modelxd /app/
WORKDIR /app
ENTRYPOINT ["/app/modelxd"]

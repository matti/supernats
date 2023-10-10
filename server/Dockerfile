FROM ubuntu:22.04

RUN apt-get update && apt-get install -y --no-install-recommends \
  curl ca-certificates unzip

RUN [ "$(uname -m)" = "aarch64" ] && arch=arm64 || arch=amd64 \
  && mkdir /ghjk && cd /ghjk \
  && curl -o nats_server.zip -Lsf "https://github.com/nats-io/nats-server/releases/download/v2.10.2/nats-server-v2.10.2-linux-${arch}.zip" \
  && unzip nats_server.zip \
  && cp "nats-server-v2.10.2-linux-${arch}/nats-server" /usr/local/bin \
  && rm -rf /ghjk

# && tar -xvof "reflex_linux_${arch}.tar.gz" \
# && mv "reflex_linux_${arch}/reflex" /usr/local/bin \
# && rm -rf /ghjk

WORKDIR /app
COPY app .

ENTRYPOINT [ "/app/entrypoint.sh" ]

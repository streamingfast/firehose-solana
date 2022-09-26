# syntax=docker/dockerfile:1.2

FROM ubuntu:20.04

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    apt-get -y install -y \
    ca-certificates libssl1.1 vim htop iotop sysstat \
    dstat strace lsof curl jq tzdata && \
    rm -rf /var/cache/apt /var/lib/apt/lists/*

RUN rm /etc/localtime && ln -snf /usr/share/zoneinfo/America/Montreal /etc/localtime && dpkg-reconfigure -f noninteractive tzdata

RUN mkdir /tmp/wasmer-install && cd /tmp/wasmer-install && \
    curl -L https://github.com/wasmerio/wasmer/releases/download/2.3.0/wasmer-linux-amd64.tar.gz | tar xzf - && \
    mv lib/libwasmer.a lib/libwasmer.so /usr/lib/ && cd / && rm -rf /tmp/wasmer-install

ADD /firesol /app/firesol

COPY tools/firesol/motd_generic /etc/
COPY tools/firesol/motd_node_manager /etc/
COPY tools/firesol/99-firehose-solana.sh /etc/profile.d/
COPY tools/firesol/scripts/* /usr/local/bin


ENTRYPOINT ["/app/firesol"]

# syntax=docker/dockerfile:1.2

FROM ghcr.io/streamingfast/firehose-core:f0391d8 as core

FROM ubuntu:20.04

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    apt-get -y install -y \
    ca-certificates libssl1.1 vim htop iotop sysstat \
    dstat strace lsof curl jq tzdata && \
    rm -rf /var/cache/apt /var/lib/apt/lists/*

RUN rm /etc/localtime && ln -snf /usr/share/zoneinfo/America/Montreal /etc/localtime && dpkg-reconfigure -f noninteractive tzdata


ADD /firesol /app/firesol

#COPY tools/docker/motd_generic /etc/
#COPY tools/docker/motd_node_manager /etc/
#COPY tools/docker/99-firehose-solana.sh /etc/profile.d/
#COPY tools/docker/scripts/* /usr/local/bin

# On SSH connection, /root/.bashrc is invoked which invokes '/root/.bash_aliases' if existing,
# so we hijack the file to "execute" our specialized bash script
#RUN echo ". /etc/profile.d/99-firehose-solana.sh" > /root/.bash_aliases

ENV PATH "$PATH:/app"

COPY --from=core /app/firecore /app/firecore

ENTRYPOINT ["/app/firesol"]


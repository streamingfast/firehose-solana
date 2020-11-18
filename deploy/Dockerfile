FROM gcr.io/eoscanada-shared-services/dubuntu:latest

RUN apt update && apt-get -y install curl ca-certificates && rm -rf /var/cache/apt/*

RUN mkdir -p /app/ && curl -Lo /app/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.2.2/grpc_health_probe-linux-amd64 && chmod +x /app/grpc_health_probe

COPY /dfusesol /app/dfusesol


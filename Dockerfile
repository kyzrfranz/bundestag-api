FROM golang:1.24 AS build
RUN apt update
RUN apt install -y ca-certificates && update-ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build-linux-amd64

FROM debian:bookworm-slim AS dockerize
RUN apt-get update && apt-get install -y \
    webp && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /app/build/buntesdach-api-amd64-linux /buntesdach-api

EXPOSE 8080
ENTRYPOINT ["/buntesdach-api"]

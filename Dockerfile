FROM golang:1.24.2-bookworm as builder

WORKDIR /src

COPY . .

RUN go install -v github.com/prometheus/promu@latest \
    && promu build -v --prefix build


FROM debian:bookworm-slim

RUN DEBIAN_FRONTEND=noninteractive; apt-get update \
    && apt-get install -qy --no-install-recommends \
        ca-certificates \
        tzdata \
        curl

COPY --from=builder /src/build/es-oneday-exporter /es-oneday-exporter

EXPOSE 9101/tcp

CMD [ "/es-oneday-exporter" ]

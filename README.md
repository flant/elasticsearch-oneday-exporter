### elasticsearch-oneday-exporter

```
usage: es-oneday-exporter [<flags>]

Authentication:

  Set OPENSEARCH_USER and OPENSEARCH_PASSWORD environment variables to authenticate with OpenSearch.

Flags:
  -h, --help                    Show context-sensitive help (also try --help-long and --help-man).
      --log.level=info          Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]
      --log.format=json         Set the log format. Valid formats: [json, text]
      --date.pattern="2006.01.02"
                                Date pattern for selecting indices.
      --telemetry.addr=":9101"  Listen on host:port.
      --telemetry.path="/metrics"
                                URL path for surfacing collected metrics.
      --address="http://localhost:9200"
                                OpenSearch node to use. Do not specify username:password in this option; use OPENSEARCH_USER and OPENSEARCH_PASSWORD environment variables instead.
      --ca-cert=""              Path to PEM file that contains trusted Certificate Authorities for the OpenSearch connection.
      --client-cert=""          Path to PEM file that contains the corresponding cert for the private key to connect to OpenSearch.
      --client-key=""           Path to PEM file that contains the private key for client auth when connecting to OpenSearch.
      --insecure                Skip SSL verification when connecting to OpenSearch.
      --project=PROJECT         Project name
      --repository=REPOSITORY   Repository name
      --version                 Show application version.
```
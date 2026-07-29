# ArtificialFlow connectors

Official connectors are external workers that implement a documented job type and payload contract. Prefer a few hardened connectors over a marketplace.

## Starter set

| Connector | Job type | Status |
| :--- | :--- | :--- |
| HTTP outbound | `io.artificialflow.connector.http` | Shipped (`connectors/http`) |
| Kafka publish | `io.artificialflow.connector.kafka` | Shipped (`connectors/kafka`) |
| Email | `io.artificialflow.connector.email` | Shipped (`connectors/email`) |
| Slack/webhook | `io.artificialflow.connector.webhook` | Shipped (`connectors/webhook`) |
| S3 | `io.artificialflow.connector.s3` | Shipped (`connectors/s3`) |

Shared helpers live in `connectors/internal/common`.

## Common env

| Variable | Default | Notes |
| :--- | :--- | :--- |
| `ARTIFICIALFLOW_BASE_URL` | `http://localhost:9100/api` | Gateway API base |
| `ARTIFICIALFLOW_TOKEN` | _(required)_ | Bearer token with job activate/complete |
| `WORKER_NAME` | per-connector default | Worker identity string |

## Run examples

```bash
export ARTIFICIALFLOW_TOKEN=...

# HTTP — see http/README.md
go run ./connectors/http

# Webhook / Slack incoming webhook URL
WEBHOOK_CONNECTOR_ALLOWED_HOSTS=hooks.slack.com go run ./connectors/webhook
# Instance vars: webhookUrl, payload?, webhookToken?

# Kafka publish
KAFKA_BROKERS=localhost:9092 go run ./connectors/kafka
# Instance vars: kafkaTopic, kafkaKey?, kafkaValue?|payload?

# Email (SMTP)
SMTP_HOST=smtp.example.com SMTP_PORT=587 SMTP_USERNAME=u SMTP_PASSWORD=p SMTP_FROM=noreply@example.com \
  go run ./connectors/email
# Instance vars: emailTo, emailSubject, emailBody?

# S3-compatible PUT (AWS or MinIO)
S3_ENDPOINT=https://s3.amazonaws.com S3_REGION=us-east-1 \
  S3_ACCESS_KEY=... S3_SECRET_KEY=... go run ./connectors/s3
# Instance vars: s3Bucket, s3Key, s3Body?, contentType?
```

## HTTP connector

See [http/README.md](http/README.md).

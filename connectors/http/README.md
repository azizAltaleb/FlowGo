# HTTP outbound connector

Job type: `io.artificialflow.connector.http`

## Variables / job custom headers (instance context)

| Field | Required | Description |
| :--- | :--- | :--- |
| `url` | yes | Absolute HTTPS/HTTP URL |
| `method` | no | Default `POST` |
| `headers` | no | Object of string headers |
| `body` | no | JSON-serializable body |
| `timeoutMs` | no | Default `10000` |

## Output variables

| Field | Description |
| :--- | :--- |
| `httpStatus` | Response status code |
| `httpBody` | Response body string (truncated) |
| `httpOk` | `true` when 2xx |

## Run

```bash
export ARTIFICIALFLOW_BASE_URL=http://localhost:9100/api
export ARTIFICIALFLOW_TOKEN=...
go run ./connectors/http
```

Security: set `HTTP_CONNECTOR_ALLOWED_HOSTS` (comma-separated hostnames) in every non-dev environment. Empty allowlist permits any host and is **evaluation-only**—treat missing allowlist as a release blocker for production.

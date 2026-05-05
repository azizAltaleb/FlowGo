# Contributing to Workflow Engine

Thank you for your interest in contributing! We want to make this project a robust open-source solution for workflow orchestration.

## Getting Started

1.  **Fork the repository** on GitHub.
2.  **Clone your fork** locally.
3.  **Start the environment**:
    ```bash
    make up
    ```
    This starts the external IAM deployment using `docker-compose.external-iam.yml`.
    No `.env` file is required. To use the bundled ZITADEL deployment instead, run:

    ```bash
    make up-zitadel
    ```

    Debezium connector bootstrap is handled automatically by `sync-worker` startup. Manual recovery remains available via `make init-connector`.

## Development Workflow

### Backend (Go)
The backend services are located in `backend/`.
- **Requirements**: Go 1.24+
- **Command Service**: `backend/services/workflow-command`
- **Query Service**: `backend/services/workflow-query`
- **Sync Worker**: `backend/services/sync-worker`

To rebuild the backend after changes:
```bash
make up
# or:
make up-zitadel
```

Validate profile configs quickly:
```bash
make smoke-profiles
```

### Frontend (React)
Located in `frontend/`.
To rebuild the frontend after changes:
```bash
make up
# or:
make up-zitadel
```

## Running Tests

We have a suite of shell scripts to verify functionality:
```bash
./demo.sh               # Runs a full deployment & execution cycle
```

CI/Security workflows:

- Main CI pipeline: `.github/workflows/ci.yml`
- Security/dependency scan pipeline: `.github/workflows/security.yml`

## Pull Request Process

1.  Create a feature branch (`git checkout -b feature/amazing-feature`).
2.  Commit your changes.
3.  Push to the branch.
4.  Open a Pull Request.

## Coding Standards

- **Go**: Follow standard Go conventions (`gofmt`, `go vet`).
- **Commits**: Use clear, descriptive commit messages.

## Architecture and Contract Guidelines

To keep the OSS engine maintainable as it grows, follow these boundaries:

1. **Model layers must stay separated**
   - Domain/persistence models stay in engine/repository boundaries.
   - Public API payloads go through DTO/mapper layers.
   - Avoid returning persistence structs directly from handlers.
   - Keep mapping rules centralized in mapper/adapter modules (do not scatter ad-hoc conversions in handlers).

   Allowed dependency direction:
   - `domain/repository -> application -> interfaces/http(dtos+mappers)`
   - `interfaces/http` may depend on `application` abstractions, but should not persist raw storage models directly.

2. **Worker API compatibility is explicit**
   - Preserve backward compatibility for worker REST contract (`/jobs/*`).
   - If you add or change worker behavior, update capabilities and protocol docs.

3. **Event evolution must remain additive when possible**
   - Prefer adding fields over reinterpreting/removing existing fields.
   - Document behavior changes in ADRs/release notes.

4. **Contract tests are required for boundary changes**
   - For worker contract changes, run:
     - `go test ./backend/services/workflow-command/internal/interfaces/http -run 'Idempotency|Protocol|Capabilities' -count=1`
     - `go test ./backend/libs/worker -count=1`
   - For query mapping/state normalization changes, run:
     - `go test ./backend/services/workflow-query/internal/interfaces/http -run 'NormalizeInstanceStateFilter|MapInstanceStatus|GetInstance_ReturnsProjectedResponse|SearchInstances_RunningStateNormalizedToActive' -count=1`

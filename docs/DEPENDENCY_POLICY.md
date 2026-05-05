# Dependency Policy

## Goals

- Keep dependencies current enough for security support.
- Prefer maintained libraries with clear licenses.
- Avoid adding runtime dependencies without a clear owner and validation path.

## Review Requirements

New dependencies should be reviewed for:

- License compatibility with MIT distribution.
- Maintenance activity.
- Known vulnerabilities.
- Transitive dependency risk.
- Runtime footprint and image impact.

## Automation

Dependabot tracks:

- Go modules.
- Frontend npm packages.
- Node.js SDK npm packages.
- GitHub Actions.
- Docker base images.

Security workflows scan Go, npm, and filesystem dependencies.

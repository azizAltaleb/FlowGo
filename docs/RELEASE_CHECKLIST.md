# Release Checklist

Use this checklist before publishing a Workflowsa public release.

## 1. Repository readiness

- Confirm the release is cut from the real Git repository, not an exported workspace.
- Confirm `.git` history is clean and the default branch has branch protection.
- Enable required status checks for CI, security, CodeQL, and release dry-runs.
- Enable Dependabot alerts, secret scanning, and code scanning.
- Add repository topics: `bpmn`, `workflow-engine`, `workflow-automation`, `golang`, `typescript`, `react`, `docker`, `oidc`, `zitadel`, `sdk`.

## 2. Secret and dependency checks

```bash
gitleaks detect --source . --redact
npm --prefix frontend audit --audit-level=high
npm --prefix clients/nodejs-sdk audit --audit-level=high
go test ./backend/... -count=1
```

Do not publish if any real `.env`, PAT, ZITADEL token, private key, client secret, or local credential is present in tracked files or release artifacts.

## 3. Local validation

```bash
make smoke-profiles
make smoke-release-profiles
npm --prefix frontend ci
npm --prefix frontend run lint
npm --prefix frontend test
npm --prefix frontend run build
npm --prefix clients/nodejs-sdk ci
npm --prefix clients/nodejs-sdk test
npm --prefix clients/nodejs-sdk pack --dry-run
go test ./backend/... -count=1
```

If Helm is installed:

```bash
helm lint ./charts/workflowsa
helm template workflowsa ./charts/workflowsa -f ./charts/workflowsa/values-external-iam.yaml >/tmp/workflowsa-external.yaml
helm template workflowsa ./charts/workflowsa -f ./charts/workflowsa/values-internal-iam.yaml >/tmp/workflowsa-internal.yaml
```

## 4. Docker Hub setup

Create or verify these Docker Hub repositories:

- `workflowsa/workflow-command`
- `workflowsa/workflow-runtime`
- `workflowsa/workflow-query`
- `workflowsa/sync-worker`
- `workflowsa/frontend`

Add GitHub Actions secrets:

- `DOCKERHUB_USERNAME`
- `DOCKERHUB_TOKEN`

## 5. npm setup

- Confirm the `@workflowsa` npm scope is available and owned by the release maintainers.
- Confirm `@workflowsa/nodejs-sdk` can be published publicly.
- Add GitHub Actions secret `NPM_TOKEN`.

## 6. Release candidate

Create an RC tag first:

```bash
git tag -s v0.1.0-rc.1 -m "Workflowsa v0.1.0-rc.1"
git push origin v0.1.0-rc.1
```

Verify release workflows produce signed images, SBOM/provenance attestations, and the npm package.

## 7. Published image smoke

After images are pushed, validate the release override:

```bash
WORKFLOWSA_IMAGE_TAG=v0.1.0-rc.1 make smoke-release-profiles
WORKFLOWSA_IMAGE_TAG=v0.1.0-rc.1 make up-zitadel-release
```

Open:

- Workflowsa: <http://localhost:9100>
- ZITADEL: <http://localhost:9180>

Sign in with local development credentials `admin` / `admin` and run the SDK smoke test with a generated SDK token.

## 8. Final release

```bash
git tag -s v0.1.0 -m "Workflowsa v0.1.0"
git push origin v0.1.0
```

Publish GitHub release notes with:

- Changelog summary.
- Known limitations.
- Docker image references.
- npm package version.
- SBOM/provenance/signing notes.
- Upgrade and security notes.

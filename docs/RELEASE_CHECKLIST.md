# Release Checklist

Use this checklist before publishing a FlowGo public release.

## 1. Repository readiness

- Confirm the release is cut from the real Git repository, not an exported workspace.
- Confirm `.git` history is clean and the default branch has branch protection.
- Enable required status checks for CI, security, CodeQL, and release dry-runs.
- Enable Dependabot alerts, secret scanning, and code scanning.
- Add repository topics: `bpmn`, `workflow-engine`, `workflow-automation`, `golang`, `typescript`, `react`, `docker`, `oidc`, `zitadel`, `sdk`.
- Open the GitHub Actions tab and confirm workflows can start jobs. If runs fail immediately with `startup_failure` and zero jobs, resolve any account, billing, or Actions enablement banner before release validation.

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
actionlint -color=false
make smoke-profiles
make smoke-release-profiles
make validate-helm
npm --prefix frontend ci
npm --prefix frontend run lint
npm --prefix frontend test
npm --prefix frontend run build
npm --prefix clients/nodejs-sdk ci
npm --prefix clients/nodejs-sdk test
npm --prefix clients/nodejs-sdk run validate:package
(cd clients/nodejs-sdk && npm pack --dry-run)
(cd clients/nodejs-sdk && npm sbom --sbom-format cyclonedx --omit dev >/tmp/flowgo-nodejs-sdk-sbom.cdx.json)
make release-dry-run
go test ./backend/... -count=1
```

`make validate-helm` always parses chart values YAML and runs `helm lint`/`helm template` when Helm is installed. To run Helm directly:

```bash
helm lint ./charts/flowgo
helm template flowgo ./charts/flowgo -f ./charts/flowgo/values-external-iam.yaml >/tmp/flowgo-external.yaml
helm template flowgo ./charts/flowgo -f ./charts/flowgo/values-internal-iam.yaml >/tmp/flowgo-internal.yaml
```

## 4. Docker Hub setup

Create or verify these Docker Hub repositories:

- `flowgo/workflow-command`
- `flowgo/workflow-runtime`
- `flowgo/workflow-query`
- `flowgo/sync-worker`
- `flowgo/frontend`

Add GitHub Actions secrets:

- `DOCKERHUB_USERNAME`
- `DOCKERHUB_TOKEN`

Use GitHub repository settings: Settings > Secrets and variables > Actions > Repository secrets.

## 5. npm setup

- Confirm the `@flowgo` npm scope is available and owned by the release maintainers.
- Confirm `@flowgo/nodejs-sdk` can be published publicly.
- Add GitHub Actions secret `NPM_TOKEN`.
- Use GitHub repository settings: Settings > Secrets and variables > Actions > Repository secrets.

## 6. Release candidate

Create an RC tag first:

```bash
git tag -s v0.1.0-rc.1 -m "FlowGo v0.1.0-rc.1"
git push origin v0.1.0-rc.1
```

Verify release workflows produce signed images, SBOM/provenance attestations, and the npm package.

## 7. Published image smoke

After images are pushed, validate the release override:

```bash
FLOWGO_IMAGE_TAG=v0.1.0-rc.1 make smoke-release-profiles
FLOWGO_IMAGE_TAG=v0.1.0-rc.1 make up-zitadel-release
```

Open:

- FlowGo: <http://localhost:9100>
- ZITADEL: <http://localhost:9180>

Sign in with local development credentials `admin` / `admin` and run the SDK smoke test with a generated SDK token.

## 8. Final release

```bash
git tag -s v0.1.0 -m "FlowGo v0.1.0"
git push origin v0.1.0
```

Publish GitHub release notes with:

- Changelog summary.
- Known limitations.
- Docker image references.
- npm package version.
- SBOM/provenance/signing notes.
- Upgrade and security notes.

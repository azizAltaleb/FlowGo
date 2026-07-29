# Release Checklist

Use this checklist before publishing an ArtificialFlow public release.

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

For bundled ZITADEL releases:

- Confirm new SDK clients receive browser-generated private-key profiles and no PAT.
- Confirm JWT Profile exchange, project audience, `artificialflow client`, SDK refresh, overlapping key rotation, and key revocation.
- Confirm legacy PAT issuance/rotation defaults remain disabled and inventory/revocation still works.
- Confirm UAT reports, screenshots, traces, and logs contain no private key, assertion, access token, PAT, or authorization header.

## 3. Local validation

```bash
actionlint -color=false
node scripts/validate_legacy_branding.mjs
node scripts/validate_release_version.mjs
node scripts/validate_release_workflows.mjs
make test-release-suite
# With a healthy stack (after make up-zitadel):
make test-all
make test-all-functionality
make test-dast
make test-uat-mega-bpmn   # needs ARTIFICIALFLOW_TOKEN
make test-perf            # needs AUTH_TOKEN + WORKFLOW_ID
make smoke-profiles
make smoke-release-profiles
bash scripts/validate_compose_identities.sh
make validate-helm
npm --prefix frontend ci
npm --prefix frontend run lint
npm --prefix frontend test
npm --prefix frontend run build
npm --prefix clients/nodejs-sdk ci
npm --prefix clients/nodejs-sdk test
npm --prefix clients/nodejs-sdk run validate:package
(cd clients/nodejs-sdk && npm pack --dry-run)
bash scripts/validate_nodejs_sdk_install.sh
bash scripts/validate_generated_proto.sh
(cd clients/nodejs-sdk && npm sbom --sbom-format cyclonedx --omit dev >/tmp/artificialflow-nodejs-sdk-sbom.cdx.json)
bash scripts/validate_migration_release.sh
make release-dry-run
go test ./backend/... -count=1
```

`make validate-helm` always parses chart values YAML and runs `helm lint`/`helm template` when Helm is installed. To run Helm directly:

```bash
helm lint ./charts/artificialflow
helm template artificialflow ./charts/artificialflow -f ./charts/artificialflow/values-external-iam.yaml >/tmp/artificialflow-external.yaml
helm template artificialflow ./charts/artificialflow -f ./charts/artificialflow/values-internal-iam.yaml >/tmp/artificialflow-internal.yaml
```

## 4. Docker Hub setup

Create or verify these canonical Docker Hub repositories:

- `artificialflow/workflow-command`
- `artificialflow/workflow-runtime`
- `artificialflow/workflow-query`
- `artificialflow/sync-worker`
- `artificialflow/frontend`

For the one-release compatibility window, also verify the corresponding five
repositories under `azizaltaleb`. Each canonical/legacy pair must report the
same manifest digest.

Add GitHub Actions secrets:

- `DOCKERHUB_USERNAME`
- `DOCKERHUB_TOKEN`

Use GitHub repository settings: Settings > Secrets and variables > Actions > Repository secrets.

`DOCKERHUB_USERNAME` is the authenticating Docker Hub user, which must be able
to push to both namespaces; it is not merely the organization name.
`DOCKERHUB_TOKEN` must be a write-capable personal access token for that same
user, not an account password. Verify access to all ten repositories before
tagging to reduce partial publication risk.

## 5. npm setup

- Confirm the `@artificialflow` npm scope is available and owned by the release maintainers.
- Confirm `@artificialflow/nodejs-sdk` can be published publicly.
- Add GitHub Actions secret `NPM_TOKEN`.
- Use GitHub repository settings: Settings > Secrets and variables > Actions > Repository secrets.
- Confirm npm trusted publishing or the automation token is authorized for both packages. Both publish steps use GitHub-hosted OIDC provenance.

## 6. Release candidate

Create an RC tag first:

```bash
git tag -s v1.0.0-rc.1 -m "ArtificialFlow v1.0.0-rc.1"
git push origin v1.0.0-rc.1
```

Before pushing it, run
`node scripts/validate_release_version.mjs --tag v1.0.0-rc.1`. The package,
lockfile, chart, Compose defaults, changelog, and documented release version
must already match the tag.

Verify release workflows produce signed images, SBOM/provenance attestations,
and both npm packages. Prerelease SDK versions publish with dist-tag `next`;
stable versions publish with `latest`. The deprecated wrapper job starts only
after the canonical publish job succeeds.

Tag pushes and publish-enabled manual runs must start the top-level `Release`
workflow. It calls the reusable image workflow first and starts the npm workflow
only after all five candidate builds pass Trivy, promotion digest checks,
canonical/legacy signature verification, and complete-set verification. The
image workflow publishes deterministic candidate references before scanning and
promotes only the scanned digest to final tags, so a retry can reuse the
candidate and accepts an existing final tag only when its digest matches.
Direct `Release Docker Images` manual runs remain available for image-only
validation or publication. Direct `Release Node.js SDK` manual runs are
validation-only; a publish-enabled direct run is refused because it has no
image-workflow dependency.

## 7. Published image smoke

After images are pushed, validate the release override:

```bash
ARTIFICIALFLOW_IMAGE_TAG=v1.0.0-rc.1 make smoke-release-profiles
ARTIFICIALFLOW_IMAGE_TAG=v1.0.0-rc.1 make up-zitadel-release
```

Open:

- ArtificialFlow: <http://localhost:9100>
- ZITADEL: <http://localhost:9180>

Sign in with local development credentials `admin` / `admin` and run the SDK smoke test with a generated SDK token.

## 8. Final release

```bash
git tag -s v1.0.0-rc.1 -m "ArtificialFlow v1.0.0-rc.1"
git push origin v1.0.0-rc.1
```

Publish GitHub release notes with:

- Changelog summary.
- Known limitations.
- Docker image references.
- npm package version.
- SBOM/provenance/signing notes.
- Upgrade and security notes.

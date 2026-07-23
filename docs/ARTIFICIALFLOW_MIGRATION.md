# Migrating from FlowGo to ArtificialFlow

This guide covers the one-release compatibility migration from the former
product identity to ArtificialFlow. Fresh installations must use only
ArtificialFlow identifiers. Existing installations may use the documented
legacy readers for one transition release while operators migrate configuration
and persistent state.

The compatibility release reads canonical identifiers first, reads an old
identifier only as a fallback, normalizes values internally, and writes or emits
canonical identifiers. It does not automatically rename live infrastructure,
delete data, move a repository, publish packages, or retag images.

## Migration map

- Git repository and Go module:
  `github.com/artificialflow/artificialflow`
- npm package: `@artificialflow/nodejs-sdk`
- temporary npm wrapper: `@flowgo/nodejs-sdk`
- Docker repositories: `artificialflow/workflow-command`,
  `artificialflow/workflow-runtime`, `artificialflow/workflow-query`,
  `artificialflow/sync-worker`, and `artificialflow/frontend`
- BPMN namespace: `http://artificialflow.io/schema/1.0/bpmn`, normally using the
  `artificialflow` prefix
- IAM roles: `artificialflow admin`, `artificialflow modeler`, and
  `artificialflow client`
- runtime configuration: `ARTIFICIALFLOW_*`
- acting-user headers: `X-ArtificialFlow-Acting-*`
- Compose project, Helm chart, image namespace, search prefix, connector,
  consumer, and state defaults: `artificialflow`

The old gRPC service path, BPMN namespace, standard IAM roles, selected
configuration variables, browser pending-sync keys, job type, search prefix,
metrics, and deployment identity overrides remain readable only for the
transition window.

## 1. Preflight, inventory, and backup

Do not start with a search-and-replace against a running deployment.

1. Record the deployed Git commit, release tag, image references and immutable
   digests, Compose project or Helm release, chart values, replica counts,
   environment, Secrets, ConfigMaps, ingress, and DNS.
2. Inventory Postgres roles and grants, databases, Debezium connector and task
   status, replication slots and LSNs, Kafka topics and consumer offsets,
   Elasticsearch/OpenSearch indices and aliases, PVCs/volumes, ZITADEL project,
   application and role IDs, and browser client IDs/redirect URIs.
3. Stop introducing new old-brand configuration. Keep a copy of the exact old
   configuration as the rollback configuration.
4. Take tested backups:
   - a Postgres backup plus role, ownership, grant, publication, and slot data;
   - an Elasticsearch/OpenSearch snapshot;
   - Kafka Connect configuration and consumer-offset capture;
   - all application and ZITADEL volumes/PVCs and Secrets;
   - ZITADEL bootstrap state and generated client files.
5. Test restore procedures in an isolated environment. A backup without a
   verified restore is not a release gate.
6. Run all migration scripts without `--apply`. They default to dry-run and
   must not mutate the deployment.

Detailed persistent-data commands and checkpoints are in
[Persistent data rename migration](PERSISTENT_DATA_RENAME_MIGRATION.md).

## 2. GitHub repository and Go imports

The repository transfer itself is an administrator operation outside this
codebase. Before changing remotes, verify that the destination repository has
the complete commit graph, tags, issues, releases, branch rules, environments,
Actions secrets, webhooks, security settings, and package permissions.

Update a clone after the transfer:

```bash
git remote -v
git remote set-url origin https://github.com/artificialflow/artificialflow.git
git fetch --all --tags
git remote -v
```

Update Go consumers to the canonical module:

```go
import workflowapi "github.com/artificialflow/artificialflow/backend/api/v1/go"
```

Then run `go mod tidy` in the consuming repository and test against the exact
release tag. Repository redirects are a convenience, not the long-term import
contract. New code must not add the old module path.

## 3. Node.js SDK and npm

Install the canonical package at the same version as the server release:

```bash
npm uninstall @flowgo/nodejs-sdk
npm install @artificialflow/nodejs-sdk@0.3.0
```

Use `ArtificialFlowClient`, `ArtificialFlowApiError`, and
`ArtificialFlow*Options`. Deprecated `FlowGo*` exports remain aliases for one
release. The temporary `@flowgo/nodejs-sdk` wrapper depends on the exact
canonical version and re-exports it; it is for applications that cannot change
the package name in the same rollout.

Release candidates use the npm `next` dist-tag. Stable releases use `latest`.
The canonical package must be published and install-tested before its wrapper.
Never publish a wrapper version whose exact canonical dependency is
unavailable.

Validate locally:

```bash
npm --prefix clients/nodejs-sdk ci
npm --prefix clients/nodejs-sdk test
npm --prefix clients/nodejs-sdk run validate:package
bash scripts/validate_nodejs_sdk_install.sh
```

## 4. Docker images

Pin an immutable digest where possible. During the compatibility release, each
canonical image and its `azizaltaleb/*` alias must be tags on the same
multi-platform build and must resolve to the same manifest digest.

```bash
docker buildx imagetools inspect artificialflow/workflow-command:v0.3.0
docker buildx imagetools inspect azizaltaleb/workflow-command:v0.3.0
```

Repeat for all five images and compare manifest digests. Verify signatures,
SBOMs, provenance, and vulnerability scan results before deployment. New
deployments and all documentation should use the `artificialflow/*` references.

The top-level release workflow waits for the complete reusable image workflow
before npm can start. Each image is built once to a deterministic candidate,
scanned by digest, and only then promoted to canonical and compatibility final
tags. Publishing verifies equal digests, both repository signatures, and the
complete five-image set. A no-publish dry run validates the workflow structure
but cannot prove remote digest or signature state.

## 5. BPMN models

Existing models using `http://flowgo.com/schema/1.0/bpmn`, the `flowgo` prefix,
or supported unqualified attributes remain importable. When both namespaces
supply the same value, the ArtificialFlow value wins. Saving or exporting a
model emits only the ArtificialFlow namespace and keys.

Recommended migration:

1. Back up the original BPMN XML.
2. Import it into the compatibility release.
3. Review assignments, service-task types, decision references, message
   correlation, timers, errors, and extension properties.
4. Export the model and confirm that only the canonical URI and prefix remain.
5. Deploy the exported copy to a non-production environment and execute every
   affected path before replacing the original definition.

Do not rewrite XML with a plain text replacement: namespace-aware parsing and
canonical precedence are required.

## 6. IAM, roles, tokens, and ZITADEL

External IAM providers must define and emit the three canonical roles. During
the transition, the server canonicalizes old standard-role claims before
authorization and API/UI output. Custom roles are preserved.

For bundled ZITADEL, keep immutable project and application IDs, client IDs,
keys, PATs, issuer settings, and redirect URIs. The bootstrap reads its saved
state first, reuses old project/application display names when necessary,
creates canonical roles, copies standard-role assignments, preserves custom
roles, and verifies that a canonical administrator remains.

Before approving an IAM upgrade, test:

- fresh bootstrap and repeated idempotent bootstrap;
- upgrade using saved old bootstrap state;
- browser login with a canonical administrator;
- a token carrying each old standard role;
- canonical role claims and custom roles;
- machine-client JWT Profile exchange and API audience;
- old PAT inventory/revocation, with new PAT creation and rotation still
  disabled by default;
- key creation, overlapping rotation, revocation, and SDK refresh;
- unchanged client IDs, redirect URIs, issuer, and project audience.

Never log tokens, private keys, assertions, authorization headers, PATs, or
client secrets. Do not delete old role definitions until no retained token or
assignment depends on them.

## 7. Headers, environment, and browser state

Move configuration to `ARTIFICIALFLOW_*`. Where a documented fallback exists,
the canonical variable has precedence even if both are present. Resolve
conflicts before the next release; do not rely on fallback order indefinitely.

SDK acting-user requests must emit only:

```text
X-ArtificialFlow-Acting-Subject
X-ArtificialFlow-Acting-Username
X-ArtificialFlow-Acting-Email
X-ArtificialFlow-Acting-Name
X-ArtificialFlow-Acting-Roles
```

The HTTP server and CORS configuration accept the old header family for one
release, preferring canonical values on conflict.

The frontend reads `window.__ARTIFICIALFLOW_RUNTIME_CONFIG__` first and the old
runtime object second. Pending workflow and instance synchronization values are
moved from old session-storage keys to `artificialflow.*` on first read, then
the old keys are removed. Test login and pending-navigation behavior with both a
clean browser profile and an upgraded profile.

## 8. Persistent data and streaming identifiers

The migration scripts are intentionally separate because these systems have
different consistency and rollback requirements:

```bash
bash scripts/test_persistent_migrations.sh
scripts/migrate_es_prefix.sh
scripts/migrate_streaming_identifiers.sh
scripts/migrate_database_and_state.sh
```

The first invocation of each migration command is a dry run. Apply only after
the confirmations in
[Persistent data rename migration](PERSISTENT_DATA_RENAME_MIGRATION.md).

Required safety properties:

- user-task reads, activation, completion, inbox, and history include canonical
  and old job types while new jobs use `artificialflow:userTask`;
- search queries use canonical indices first and fall back only for a missing
  canonical index/document, never for arbitrary cluster errors;
- canonical and old outbox metrics expose the same snapshot during the window;
- the sync worker rejects simultaneous old and canonical Debezium topic
  families by default;
- a canonical connector is not started while the old connector is running;
- connector offsets, slot LSN, topic cutover, consumer lag, counts, and complete
  ID digests are recorded and verified;
- old indices, roles, slots, topics, groups, state directories, volumes, and
  claims remain available for rollback.

## 9. Docker Compose upgrade

Fresh Compose deployments use the `artificialflow` project and canonical named
volumes. Existing deployments must discover and explicitly set their exact old
project, volume, database, search, connector, group, topic, slot, and state
identifiers before rendering.

Validate both identities without starting services:

```bash
docker compose -f docker-compose.zitadel.yml config --quiet
docker compose -f docker-compose.zitadel.yml -f docker-compose.release.yml config --quiet
bash scripts/validate_compose_identities.sh
```

A wrong volume override creates empty storage and can look like data loss. Do
not use `docker compose down -v`, `make clean-*`, or any volume-deleting command
during the migration or rollback window.

## 10. Helm upgrade and resource stability

Fresh installs use release `artificialflow` and the canonical chart defaults.
An in-place upgrade of an old release must include
`charts/artificialflow/values-legacy-persistent-identifiers.yaml`, corrected
from the live manifest for any nonstandard name:

```bash
helm template flowgo ./charts/artificialflow \
  -f ./charts/artificialflow/values-internal-iam.yaml \
  -f ./charts/artificialflow/values-legacy-persistent-identifiers.yaml \
  > /tmp/artificialflow-upgrade.yaml
```

Review the rendered diff before `helm upgrade`. Immutable selectors, Service
selectors, Secret/PVC names, existing claims, and StatefulSet retention must
remain attached to the old resources. The compatibility values and chart add
keep/retain protections; they do not migrate resource names.

For the former bundled-IAM stock deployment, the compatibility values also keep
`https://flowgo.example.com` as the application origin and
`https://iam.flowgo.example.com` as the issuer/frontend authority, including the
old ingress hosts, TLS Secrets, CORS origin, bootstrap frontend URL, and
`/flowgo/bootstrap/flowgo-frontend-client-id`. This is a login-safety boundary,
not cosmetic branding. Keep the old URL canonical until an explicit IAM
migration adds and verifies both old and new redirect URIs, post-logout redirect
URIs, and additional origins on the reused ZITADEL application. External-IAM
upgrades must preserve the provider values captured from the live release.

Moving to canonical Kubernetes resource names is a controlled blue/green
cutover. Back up and clone state, quiesce old writers, install the canonical
release behind a separate routing boundary, verify it, switch traffic, and keep
the old release unchanged for rollback. Never uninstall the old release before
all retained Secrets and PVCs are independently verified.

Run:

```bash
bash scripts/validate_helm.sh
```

This lints and renders fresh external/internal IAM installs plus old-identity
upgrades and checks exact resource identities, selectors, images, and retention.

## 11. Release order and validation

Use a release candidate before stable:

1. Tag one verified commit with a version matching npm, lockfile, chart,
   Compose defaults, changelog, and documented defaults.
2. Build canonical images once, attach compatibility repository tags to the
   same manifests, then scan, verify digests, sign, and smoke-test artifacts.
3. Publish the canonical npm package with `next`, pack/install it in a clean
   consumer, then publish and install-test the exact-version wrapper.
4. Run fresh-install and upgrade smoke tests against published artifacts.
5. Promote the same commit to stable only after every migration gate passes.

Repository preflight:

```bash
node scripts/validate_legacy_branding.mjs
node scripts/validate_release_version.mjs --tag v0.3.0
node scripts/validate_release_workflows.mjs
bash scripts/validate_migration_release.sh
RUN_DOCKER_BUILDS=false bash scripts/release_dry_run.sh
```

The no-build release dry run does not publish, transfer, migrate, or contact a
live registry. Published-artifact digest, signature, IAM, connector, and
end-to-end checks remain deployment gates.

## 12. Verification and rollback

Before resuming production writes, verify:

- Git and Go consumers use the canonical repository/module;
- canonical and wrapper npm installs load the same SDK implementation;
- every image is pinned and canonical/alias digests match;
- old, canonical, and mixed BPMN models execute and export canonically;
- browser and machine login, roles, audiences, keys, and token rotation work;
- canonical values win for conflicting header/environment/runtime inputs;
- upgraded browser storage moves once and old keys disappear;
- both user-task types can be queried, claimed, completed, and reported;
- Postgres/search counts and complete ID sets match;
- exactly one connector and consumer generation is active, offsets/LSN are
  continuous, and lag converges;
- canonical and old metric values match;
- Compose volumes or Helm resources remain attached to the intended data;
- command/query health, authenticated identity, workflow start, runtime
  progression, and query projection all pass.

Rollback in reverse order. Stop canonical writers and consumers first, restore
traffic to the old release, restore the recorded configuration and image
digests, and reattach retained state. If the canonical deployment accepted
writes, reconcile or restore data according to the tested recovery plan before
reopening the old stack. Never run old and canonical connector/writer
generations concurrently.

## 13. Compatibility removal criteria

The compatibility window lasts one release. Removal is a separate, reviewed
change in the following release and is allowed only when all criteria are met:

- repository validation reports no unapproved old-brand references;
- configuration inventories show no old environment variables or headers;
- access logs or equivalent telemetry show zero old gRPC paths, headers, BPMN
  imports, roles/tokens, job types, search fallbacks, browser keys, connector
  identifiers, and metric consumers for the documented observation window;
- every maintained SDK consumer uses the canonical package and symbols;
- every model has been re-exported or verified;
- ZITADEL/external-IAM assignments and unexpired tokens use canonical roles;
- search, Kafka/Debezium, database, Compose, Helm, volumes/PVCs, and state paths
  have completed their approved migrations;
- dashboards and alerts consume canonical metrics;
- fresh-install, upgrade, rollback, and published-artifact suites pass;
- backups, old images, configuration, and data are retained for the stated
  rollback period.

If telemetry is unavailable for a surface, use an explicit inventory and a
longer observation window; absence of evidence is not evidence that a reader is
safe to remove. Update the allowlist as each compatibility reader is deleted so
that it cannot be accidentally reintroduced.

#!/usr/bin/env bash
set -euo pipefail

MODE="dry-run"
CONFIRM=""
ADMIN_PG_DSN="${ADMIN_PG_DSN:-}"
OLD_DB_ROLE="${OLD_DB_ROLE:-flowgo}"
NEW_DB_ROLE="${NEW_DB_ROLE:-artificialflow}"
NEW_DB_PASSWORD="${NEW_DB_PASSWORD:-}"
STATE_SOURCE="${STATE_SOURCE:-}"
STATE_DESTINATION="${STATE_DESTINATION:-}"
STATE_UID="${STATE_UID:-10001}"
STATE_GID="${STATE_GID:-10001}"

usage() {
  cat <<'EOF'
Usage: scripts/migrate_database_and_state.sh [--apply --confirm migrate-artificialflow-state]

Dry-run is the default. Apply mode creates (but never drops) the artificialflow
database role, inherits grants from the old role, mirrors replication privilege,
and optionally copies application state to a new path while preserving metadata.
EOF
}

while (($#)); do
  case "$1" in
    --apply) MODE="apply" ;;
    --confirm)
      shift
      CONFIRM="${1:-}"
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

if [[ "${MODE}" == "dry-run" ]]; then
  cat <<EOF
DRY RUN: no database or filesystem changes were made.

Database plan:
- Inspect numeric ownership and grants for ${OLD_DB_ROLE}.
- Create ${NEW_DB_ROLE} only if absent; retain ${OLD_DB_ROLE} for rollback.
- Grant ${OLD_DB_ROLE} to ${NEW_DB_ROLE} so existing object grants remain inherited.
- Mirror LOGIN/REPLICATION capability without moving object ownership.

State plan:
- Copy STATE_SOURCE to STATE_DESTINATION with cp -a (source is retained).
- Set copied state to STATE_UID:STATE_GID after verifying the target image UID/GID.
- Compare complete file lists and SHA-256 digests before cutover.

Apply explicitly:
  ADMIN_PG_DSN=... NEW_DB_PASSWORD=... \\
  STATE_SOURCE=... STATE_DESTINATION=... STATE_UID=... STATE_GID=... \\
    $0 --apply --confirm migrate-artificialflow-state
EOF
  exit 0
fi

if [[ "${CONFIRM}" != "migrate-artificialflow-state" ]]; then
  echo "Refusing apply: pass --confirm migrate-artificialflow-state" >&2
  exit 2
fi
if [[ -z "${ADMIN_PG_DSN}" || -z "${NEW_DB_PASSWORD}" ]]; then
  echo "Refusing apply: ADMIN_PG_DSN and NEW_DB_PASSWORD are required" >&2
  exit 2
fi
command -v psql >/dev/null

old_role_csv="$(psql "${ADMIN_PG_DSN}" --no-psqlrc --tuples-only --csv \
  --command "SELECT rolname, rolcanlogin, rolreplication FROM pg_roles WHERE rolname = :'old_role';" \
  --set old_role="${OLD_DB_ROLE}")"
if [[ -z "${old_role_csv}" ]]; then
  echo "Legacy database role ${OLD_DB_ROLE} does not exist" >&2
  exit 1
fi

new_exists="$(psql "${ADMIN_PG_DSN}" --no-psqlrc --tuples-only --no-align \
  --command "SELECT 1 FROM pg_roles WHERE rolname = :'new_role';" \
  --set new_role="${NEW_DB_ROLE}")"
if [[ "${new_exists}" != "1" ]]; then
  psql "${ADMIN_PG_DSN}" --no-psqlrc --set ON_ERROR_STOP=1 \
    --set new_role="${NEW_DB_ROLE}" --set new_password="${NEW_DB_PASSWORD}" \
    --command 'CREATE ROLE :"new_role" LOGIN PASSWORD :'\''new_password'\'';'
fi

psql "${ADMIN_PG_DSN}" --no-psqlrc --set ON_ERROR_STOP=1 \
  --set old_role="${OLD_DB_ROLE}" --set new_role="${NEW_DB_ROLE}" \
  --command 'GRANT :"old_role" TO :"new_role";'
if [[ "${old_role_csv##*,}" == "t" ]]; then
  psql "${ADMIN_PG_DSN}" --no-psqlrc --set ON_ERROR_STOP=1 \
    --set new_role="${NEW_DB_ROLE}" \
    --command 'ALTER ROLE :"new_role" WITH REPLICATION;'
fi
echo "Database role checkpoint complete; ${OLD_DB_ROLE} was retained."

if [[ -n "${STATE_SOURCE}" || -n "${STATE_DESTINATION}" ]]; then
  if [[ -z "${STATE_SOURCE}" || -z "${STATE_DESTINATION}" ]]; then
    echo "STATE_SOURCE and STATE_DESTINATION must be set together" >&2
    exit 2
  fi
  if [[ ! -d "${STATE_SOURCE}" ]]; then
    echo "STATE_SOURCE is not a directory: ${STATE_SOURCE}" >&2
    exit 1
  fi
  mkdir -p "${STATE_DESTINATION}"
  cp -a "${STATE_SOURCE}/." "${STATE_DESTINATION}/"
  chown -R "${STATE_UID}:${STATE_GID}" "${STATE_DESTINATION}"

  source_manifest="$(mktemp)"
  destination_manifest="$(mktemp)"
  (
    cd "${STATE_SOURCE}"
    find . -type f -print0 | LC_ALL=C sort -z | xargs -0 shasum -a 256
  ) >"${source_manifest}"
  (
    cd "${STATE_DESTINATION}"
    find . -type f -print0 | LC_ALL=C sort -z | xargs -0 shasum -a 256
  ) >"${destination_manifest}"
  if ! cmp -s "${source_manifest}" "${destination_manifest}"; then
    rm -f "${source_manifest}" "${destination_manifest}"
    echo "State checksum comparison failed; source remains untouched" >&2
    exit 1
  fi
  rm -f "${source_manifest}" "${destination_manifest}"
  echo "State copy verified; source path was retained for rollback."
fi

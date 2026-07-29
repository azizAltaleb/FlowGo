#!/usr/bin/env node
/**
 * Mint a short-lived ArtificialFlow access token for golden-demo bake-off / k6.
 * Uses the bundled ZITADEL owner PAT inside Compose (never prints the PAT).
 *
 * Writes token to OUT_FILE (default: .local/bakeoff.token) and prints only the path.
 */
import { createPrivateKey, generateKeyPairSync, sign } from "node:crypto";
import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";

const COMPOSE = [
  "compose",
  "-f",
  "docker-compose.zitadel.yml",
  "-f",
  "docker-compose.release.yml",
];
const ISSUER = (process.env.ZITADEL_ISSUER || "http://localhost:9180").replace(/\/$/, "");
const OUT_FILE = resolve(process.env.OUT_FILE || ".local/bakeoff.token");
const ROLES = (process.env.BAKEOFF_ROLES || "artificialflow client,artificialflow modeler")
  .split(",")
  .map((r) => r.trim())
  .filter(Boolean);

function dcExec(service, ...args) {
  return execFileSync("docker", [...COMPOSE, "exec", "-T", service, ...args], {
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  }).trim();
}

function connect(ownerPat, path, payload) {
  const res = fetch(`${ISSUER}${path}`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${ownerPat}`,
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    body: JSON.stringify(payload ?? {}),
  });
  return res;
}

function encodeJson(obj) {
  return Buffer.from(JSON.stringify(obj)).toString("base64url");
}

async function main() {
  const ownerPat = dcExec("zitadel-login", "cat", "/zitadel/bootstrap/owner.pat");
  if (!ownerPat) throw new Error("owner PAT empty");

  const stateRaw = dcExec(
    "zitadel-login",
    "cat",
    "/artificialflow/bootstrap/artificialflow-zitadel.json",
  );
  const state = JSON.parse(stateRaw);
  const orgId = state.org_id || state.orgId || state.organizationId;
  const projectId = state.project_id || state.projectId;
  if (!orgId || !projectId) {
    throw new Error("bootstrap state missing org/project id");
  }

  const { privateKey, publicKey } = generateKeyPairSync("rsa", {
    modulusLength: 2048,
    publicKeyEncoding: { type: "spki", format: "pem" },
    privateKeyEncoding: { type: "pkcs8", format: "pem" },
  });
  // ZITADEL expects PKIX PEM bytes (JSON []byte → base64 of the PEM text).
  const publicKeyB64 = Buffer.from(publicKey, "utf8").toString("base64");

  const createRes = await connect(ownerPat, "/zitadel.user.v2.UserService/CreateUser", {
    organizationId: orgId,
    username: `bakeoff-${Date.now()}`,
    machine: {
      name: "bakeoff-sdk",
      description: "ephemeral bake-off client",
      accessTokenType: "ACCESS_TOKEN_TYPE_JWT",
    },
  });
  const createBody = await createRes.json();
  if (!createRes.ok) {
    throw new Error(`CreateUser failed: ${createRes.status} ${JSON.stringify(createBody)}`);
  }
  const userId = createBody.id || createBody.userId;
  if (!userId) throw new Error("CreateUser returned no id");

  const authRes = await connect(ownerPat, "/zitadel.authorization.v2.AuthorizationService/CreateAuthorization", {
    userId,
    projectId,
    organizationId: orgId,
    roleKeys: ROLES,
  });
  if (!authRes.ok) {
    const body = await authRes.text();
    throw new Error(`CreateAuthorization failed: ${authRes.status} ${body}`);
  }

  const exp = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();
  const keyRes = await connect(ownerPat, "/zitadel.user.v2.UserService/AddKey", {
    userId,
    publicKey: publicKeyB64,
    expirationDate: exp,
  });
  const keyBody = await keyRes.json();
  if (!keyRes.ok) {
    throw new Error(`AddKey failed: ${keyRes.status} ${JSON.stringify(keyBody)}`);
  }
  const keyId = keyBody.keyId || keyBody.id;
  if (!keyId) throw new Error("AddKey returned no keyId");

  const now = Math.floor(Date.now() / 1000);
  const assertion = `${encodeJson({ alg: "RS256", kid: keyId, typ: "JWT" })}.${encodeJson({
    iss: userId,
    sub: userId,
    aud: ISSUER,
    iat: now,
    exp: now + 3600,
  })}`;
  const sig = sign("RSA-SHA256", Buffer.from(assertion), createPrivateKey(privateKey)).toString("base64url");
  const jwt = `${assertion}.${sig}`;

  const tokenRes = await fetch(`${ISSUER}/oauth/v2/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "urn:ietf:params:oauth:grant-type:jwt-bearer",
      assertion: jwt,
      scope: [
        "openid",
        "urn:zitadel:iam:org:projects:roles",
        `urn:zitadel:iam:org:project:id:${projectId}:aud`,
      ].join(" "),
    }),
  });
  const tokenBody = await tokenRes.json();
  if (!tokenRes.ok || !tokenBody.access_token) {
    throw new Error(`token exchange failed: ${tokenRes.status} ${JSON.stringify(tokenBody)}`);
  }

  mkdirSync(dirname(OUT_FILE), { recursive: true });
  writeFileSync(OUT_FILE, tokenBody.access_token, { mode: 0o600 });
  console.log(OUT_FILE);
}

main().catch((err) => {
  console.error(err.message || err);
  process.exit(1);
});

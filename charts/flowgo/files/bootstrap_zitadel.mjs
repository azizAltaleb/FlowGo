import http from "node:http";
import https from "node:https";
import fs from "node:fs";
import path from "node:path";

class ZitadelError extends Error {
  constructor(status, body) {
    super(`ZITADEL request failed with status ${status}: ${JSON.stringify(body)}`);
    this.status = status;
    this.body = body;
  }
}

function env(name, fallback = "") {
  return (process.env[name] || fallback).trim();
}

function canonicalIdentity(value) {
  return String(value || "").trim().toLowerCase();
}

const ZITADEL_INTERNAL_URL = env("ZITADEL_INTERNAL_URL", "http://zitadel-api:8080").replace(/\/$/, "");
const ZITADEL_PUBLIC_URL = env("ZITADEL_PUBLIC_URL", "http://localhost:9180").replace(/\/$/, "");
const OWNER_PAT_FILE = env("ZITADEL_OWNER_PAT_FILE", "/zitadel/bootstrap/owner.pat");
const CLIENT_ID_FILE = env("FLOWGO_FRONTEND_CLIENT_ID_FILE", "/flowgo/bootstrap/flowgo-frontend-client-id");
const BOOTSTRAP_STATE_FILE = env("FLOWGO_ZITADEL_BOOTSTRAP_STATE_FILE", "/flowgo/bootstrap/flowgo-zitadel.json");
const PROJECT_NAME = env("FLOWGO_PROJECT_NAME", "FlowGo");
const FRONTEND_APP_NAME = env("FLOWGO_FRONTEND_APP_NAME", "FlowGo Frontend");
const FRONTEND_URL = env("FLOWGO_FRONTEND_URL", "http://localhost:9100").replace(/\/$/, "");
const ADMIN_USERNAME = env("ZITADEL_ADMIN_USERNAME", env("ZITADEL_ADMIN_LOGIN_NAME", "admin"));
const ADMIN_PASSWORD = env("ZITADEL_ADMIN_PASSWORD", "admin");
const ADMIN_GIVEN_NAME = env("ZITADEL_ADMIN_GIVEN_NAME", "admin");
const ADMIN_FAMILY_NAME = env("ZITADEL_ADMIN_FAMILY_NAME", "admin");
const ADMIN_DISPLAY_NAME = env("ZITADEL_ADMIN_DISPLAY_NAME", "admin");
const ADMIN_EMAIL = env("ZITADEL_ADMIN_EMAIL", "admin@admin.localhost");
const ADMIN_LOGIN_NAME = env("ZITADEL_ADMIN_LOGIN_NAME", ADMIN_USERNAME);
const ADMIN_IDENTIFIERS = new Set([ADMIN_LOGIN_NAME, ADMIN_USERNAME, ADMIN_EMAIL].map(canonicalIdentity).filter(Boolean));
const LEGACY_ADMIN_IDENTIFIERS = new Set(
  env("ZITADEL_LEGACY_ADMIN_LOGIN_NAMES", "admin,zitadel-admin@zitadel.localhost,admin@admin.admin")
    .split(",")
    .map(canonicalIdentity)
    .filter((value) => value && !ADMIN_IDENTIFIERS.has(value)),
);
const PUBLIC_HOST = new URL(ZITADEL_PUBLIC_URL).host;
const ROLES = [
  ["flowgo client", "FlowGo Client"],
  ["flowgo admin", "FlowGo Admin"],
  ["flowgo viewer", "FlowGo Viewer"],
];

function log(message) {
  console.log(`[zitadel-bootstrap] ${message}`);
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function requestJson(method, requestPath, token = "", payload = undefined, expected = [200]) {
  const url = new URL(`${ZITADEL_INTERNAL_URL}${requestPath}`);
  const body = payload === undefined ? undefined : Buffer.from(JSON.stringify(payload));
  const headers = {
    Accept: "application/json",
    Host: PUBLIC_HOST,
  };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  if (body) {
    headers["Content-Type"] = "application/json";
    headers["Content-Length"] = String(body.length);
  }
  const transport = url.protocol === "https:" ? https : http;
  return new Promise((resolve, reject) => {
    const request = transport.request(
      url,
      {
        method,
        headers,
      },
      (response) => {
        const chunks = [];
        response.on("data", (chunk) => chunks.push(chunk));
        response.on("end", () => {
          const raw = Buffer.concat(chunks).toString("utf8");
          let parsed = {};
          if (raw) {
            try {
              parsed = JSON.parse(raw);
            } catch {
              parsed = { raw };
            }
          }
          if (expected.length && !expected.includes(response.statusCode)) {
            reject(new ZitadelError(response.statusCode, parsed));
            return;
          }
          resolve(parsed);
        });
      },
    );
    request.setTimeout(15000, () => request.destroy(new Error("ZITADEL request timed out")));
    request.on("error", reject);
    if (body) {
      request.write(body);
    }
    request.end();
  });
}

function connect(requestPath, token, payload = {}) {
  return requestJson("POST", requestPath, token, payload, [200]);
}

async function waitForZitadel() {
  const deadline = Date.now() + Number(env("ZITADEL_BOOTSTRAP_WAIT_SECONDS", "180")) * 1000;
  while (Date.now() < deadline) {
    try {
      await requestJson("GET", "/debug/ready", "", undefined, [200, 204]);
      return;
    } catch (error) {
      log(`waiting for ZITADEL API: ${error.message}`);
      await sleep(2000);
    }
  }
  throw new Error("ZITADEL API did not become ready in time");
}

async function waitForPat() {
  const deadline = Date.now() + Number(env("ZITADEL_BOOTSTRAP_WAIT_SECONDS", "180")) * 1000;
  while (Date.now() < deadline) {
    try {
      const stat = fs.statSync(OWNER_PAT_FILE);
      if (stat.size > 0) {
        return fs.readFileSync(OWNER_PAT_FILE, "utf8").trim();
      }
    } catch {}
    log(`waiting for owner PAT at ${OWNER_PAT_FILE}`);
    await sleep(2000);
  }
  throw new Error(`owner PAT was not generated at ${OWNER_PAT_FILE}; recreate the ZITADEL first-instance volume after enabling owner PAT generation`);
}

async function getOrgId(token) {
  const response = await requestJson("GET", "/management/v1/orgs/me", token, undefined, [200]);
  return response.org.id;
}

async function listProjects(token) {
  const response = await connect("/zitadel.project.v2.ProjectService/ListProjects", token, { pagination: { limit: "100" } });
  return response.projects || [];
}

async function ensureProject(token, orgId) {
  for (const project of await listProjects(token)) {
    if (project.name === PROJECT_NAME) {
      log(`using existing project ${PROJECT_NAME} (${project.projectId})`);
      return project.projectId;
    }
  }
  const response = await connect("/zitadel.project.v2.ProjectService/CreateProject", token, {
    organizationId: orgId,
    name: PROJECT_NAME,
    projectRoleAssertion: true,
    authorizationRequired: false,
    projectAccessRequired: false,
  });
  log(`created project ${PROJECT_NAME} (${response.projectId})`);
  return response.projectId;
}

function isAlreadyExists(error) {
  const body = JSON.stringify(error?.body || error).toLowerCase();
  return body.includes("already") || body.includes("exists") || body.includes("precondition");
}

function isNotChanged(error) {
  return JSON.stringify(error?.body || error).toLowerCase().includes("notchanged");
}

function loginPolicyPayload(policy = {}) {
  return {
    allowUsernamePassword: policy.allowUsernamePassword ?? true,
    allowRegister: false,
    allowExternalIdp: policy.allowExternalIdp ?? true,
    forceMfa: policy.forceMfa ?? false,
    passwordlessType: policy.passwordlessType || "PASSWORDLESS_TYPE_ALLOWED",
    hidePasswordReset: policy.hidePasswordReset ?? false,
    ignoreUnknownUsernames: policy.ignoreUnknownUsernames ?? false,
    passwordCheckLifetime: policy.passwordCheckLifetime || "864000s",
    externalLoginCheckLifetime: policy.externalLoginCheckLifetime || "864000s",
    mfaInitSkipLifetime: policy.mfaInitSkipLifetime || "2592000s",
    secondFactorCheckLifetime: policy.secondFactorCheckLifetime || "64800s",
    multiFactorCheckLifetime: policy.multiFactorCheckLifetime || "43200s",
    allowDomainDiscovery: policy.allowDomainDiscovery ?? true,
  };
}

async function ensureRegistrationDisabled(token) {
  const response = await requestJson("GET", "/management/v1/policies/login", token, undefined, [200]);
  const policy = response.policy || {};
  if (response.isDefault !== true && policy.isDefault !== true && policy.allowRegister !== true) {
    log("user self-registration is disabled");
    return;
  }
  const payload = loginPolicyPayload(policy);
  const method = response.isDefault === true || policy.isDefault === true ? "POST" : "PUT";
  try {
    await requestJson(method, "/management/v1/policies/login", token, payload, [200]);
  } catch (error) {
    if (method === "POST" && isAlreadyExists(error)) {
      try {
        await requestJson("PUT", "/management/v1/policies/login", token, payload, [200]);
      } catch (updateError) {
        if (!isNotChanged(updateError)) {
          throw updateError;
        }
      }
    } else if (!isNotChanged(error)) {
      throw error;
    }
  }
  log("disabled user self-registration");
}

function passwordComplexityPolicyPayload(policy = {}) {
  const minLength = Math.max(1, ADMIN_PASSWORD.length);
  return {
    minLength: String(Math.min(Number(policy.minLength || minLength), minLength)),
    hasLowercase: false,
    hasUppercase: false,
    hasNumber: false,
    hasSymbol: false,
  };
}

async function ensureAdminPasswordAccepted(token) {
  const response = await requestJson("GET", "/management/v1/policies/password/complexity", token, undefined, [200]);
  const policy = response.policy || {};
  const desired = passwordComplexityPolicyPayload(policy);
  const alreadyAllowsAdminPassword =
    Number(policy.minLength || 0) <= ADMIN_PASSWORD.length &&
    policy.hasUppercase !== true &&
    policy.hasNumber !== true &&
    policy.hasSymbol !== true;
  if (response.isDefault !== true && policy.isDefault !== true && alreadyAllowsAdminPassword) {
    log("password policy accepts default admin password");
    return;
  }
  const method = response.isDefault === true || policy.isDefault === true ? "POST" : "PUT";
  try {
    await requestJson(method, "/management/v1/policies/password/complexity", token, desired, [200]);
  } catch (error) {
    if (method === "POST" && isAlreadyExists(error)) {
      try {
        await requestJson("PUT", "/management/v1/policies/password/complexity", token, desired, [200]);
      } catch (updateError) {
        if (!isNotChanged(updateError)) {
          throw updateError;
        }
      }
    } else if (!isNotChanged(error)) {
      throw error;
    }
  }
  log("configured password policy for default admin password");
}

async function ensureRoles(token, projectId) {
  for (const [roleKey, displayName] of ROLES) {
    try {
      await connect("/zitadel.project.v2.ProjectService/AddProjectRole", token, {
        projectId,
        roleKey,
        displayName,
        group: "FlowGo",
      });
      log(`created role ${roleKey}`);
    } catch (error) {
      if (isAlreadyExists(error)) {
        log(`role already exists: ${roleKey}`);
        continue;
      }
      throw error;
    }
  }
}

async function listApplications(token) {
  const response = await connect("/zitadel.application.v2.ApplicationService/ListApplications", token, { pagination: { limit: "100" } });
  return response.applications || [];
}

async function ensureFrontendApplication(token, projectId) {
  for (const app of await listApplications(token)) {
    const oidc = app.oidcConfiguration || {};
    if (app.projectId === projectId && app.name === FRONTEND_APP_NAME && oidc.clientId) {
      log(`using existing frontend app ${FRONTEND_APP_NAME} with client ID ${oidc.clientId}`);
      return [app.applicationId, oidc.clientId];
    }
  }
  const response = await connect("/zitadel.application.v2.ApplicationService/CreateApplication", token, {
    projectId,
    name: FRONTEND_APP_NAME,
    oidcConfiguration: {
      redirectUris: [FRONTEND_URL],
      responseTypes: ["OIDC_RESPONSE_TYPE_CODE"],
      grantTypes: ["OIDC_GRANT_TYPE_AUTHORIZATION_CODE"],
      applicationType: "OIDC_APP_TYPE_USER_AGENT",
      authMethodType: "OIDC_AUTH_METHOD_TYPE_NONE",
      postLogoutRedirectUris: [FRONTEND_URL],
      version: "OIDC_VERSION_1_0",
      developmentMode: true,
      accessTokenType: "OIDC_TOKEN_TYPE_JWT",
      accessTokenRoleAssertion: true,
      idTokenRoleAssertion: true,
      idTokenUserinfoAssertion: true,
      additionalOrigins: [FRONTEND_URL],
    },
  });
  const clientId = response.oidcConfiguration?.clientId;
  if (!clientId) {
    throw new Error(`ZITADEL did not return an OIDC client ID: ${JSON.stringify(response)}`);
  }
  log(`created frontend app ${FRONTEND_APP_NAME} with client ID ${clientId}`);
  return [response.applicationId, clientId];
}

async function listUsers(token) {
  const response = await connect("/zitadel.user.v2.UserService/ListUsers", token, { pagination: { limit: "500" } });
  return response.result || [];
}

function userIdentities(user) {
  return [
    user.username,
    user.preferredLoginName,
    user.human?.email?.email,
    ...(user.loginNames || []),
  ].map(canonicalIdentity).filter(Boolean);
}

function userLoginIdentities(user) {
  return [
    user.username,
    user.preferredLoginName,
    ...(user.loginNames || []),
  ].map(canonicalIdentity).filter(Boolean);
}

function isTargetAdminLoginUser(user) {
  return user.human && userLoginIdentities(user).some((name) => ADMIN_IDENTIFIERS.has(name));
}

function isSolutionAdminUser(user) {
  return user.human && userIdentities(user).some((name) => ADMIN_IDENTIFIERS.has(name));
}

function isLegacySolutionAdminUser(user) {
  return user.human && userIdentities(user).some((name) => LEGACY_ADMIN_IDENTIFIERS.has(name));
}

async function findAdminUser(token) {
  const users = await listUsers(token);
  return users.find(isTargetAdminLoginUser) || users.find(isSolutionAdminUser);
}

async function deleteUser(token, user) {
  await connect("/zitadel.user.v2.UserService/DeleteUser", token, { userId: user.userId });
}

async function listProjectAuthorizations(token) {
  const response = await connect("/zitadel.authorization.v2.AuthorizationService/ListAuthorizations", token, { pagination: { limit: "500" } });
  return response.authorizations || [];
}

function authorizationHasAdminRole(authorization, projectId) {
  return authorization.project?.id === projectId && (authorization.roles || []).some((role) => role.key === "flowgo admin");
}

async function updateAdminUser(token, user) {
  try {
    await connect("/zitadel.user.v2.UserService/UpdateUser", token, {
      userId: user.userId,
      username: ADMIN_USERNAME,
      human: {
        profile: {
          givenName: ADMIN_GIVEN_NAME,
          familyName: ADMIN_FAMILY_NAME,
          displayName: ADMIN_DISPLAY_NAME,
          preferredLanguage: "en",
        },
        email: {
          email: ADMIN_EMAIL,
          isVerified: true,
        },
      },
    });
    log(`updated admin user login to ${ADMIN_LOGIN_NAME}`);
  } catch (error) {
    if (!isNotChanged(error)) {
      throw error;
    }
  }
}

async function ensureAdminUser(token, orgId) {
  const existingUser = await findAdminUser(token);
  if (existingUser) {
    await updateAdminUser(token, existingUser);
    log(`using existing admin user ${ADMIN_LOGIN_NAME}`);
    return existingUser.userId;
  }
  const response = await connect("/zitadel.user.v2.UserService/CreateUser", token, {
    organizationId: orgId,
    username: ADMIN_USERNAME,
    human: {
      profile: {
        givenName: ADMIN_GIVEN_NAME,
        familyName: ADMIN_FAMILY_NAME,
        displayName: ADMIN_DISPLAY_NAME,
        preferredLanguage: "en",
      },
      email: {
        email: ADMIN_EMAIL,
        isVerified: true,
      },
      password: {
        password: ADMIN_PASSWORD,
        changeRequired: false,
      },
    },
  });
  log(`created admin user ${ADMIN_LOGIN_NAME}`);
  return response.id;
}

async function cleanupExtraAdminUsers(token, projectId, retainedAdminUserId) {
  const users = await listUsers(token);
  const usersById = new Map(users.map((user) => [user.userId, user]));
  const usersToDelete = new Map();
  for (const user of users) {
    if (user.userId !== retainedAdminUserId && isLegacySolutionAdminUser(user)) {
      usersToDelete.set(user.userId, user);
    }
  }
  for (const authorization of await listProjectAuthorizations(token)) {
    const userId = authorization.user?.id;
    const user = usersById.get(userId);
    if (userId && userId !== retainedAdminUserId && user?.human && authorizationHasAdminRole(authorization, projectId)) {
      usersToDelete.set(userId, user);
    }
  }
  for (const user of usersToDelete.values()) {
    await deleteUser(token, user);
    log(`deleted extra human admin user ${user.preferredLoginName || user.username || user.human?.email?.email || user.userId}`);
  }
}

async function assignAdminRole(token, orgId, projectId, userId) {
  if (!userId) {
    log(`admin user not found, skipping role assignment: ${ADMIN_LOGIN_NAME}`);
    return;
  }
  try {
    await connect("/zitadel.authorization.v2.AuthorizationService/CreateAuthorization", token, {
      userId,
      projectId,
      organizationId: orgId,
      roleKeys: ["flowgo admin"],
    });
    log(`assigned flowgo admin to ${ADMIN_LOGIN_NAME}`);
  } catch (error) {
    if (isAlreadyExists(error)) {
      log(`admin authorization already exists for ${ADMIN_LOGIN_NAME}`);
      return;
    }
    throw error;
  }
}

function writeText(filePath, value) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  const tempPath = `${filePath}.tmp`;
  fs.writeFileSync(tempPath, `${value}\n`, "utf8");
  fs.renameSync(tempPath, filePath);
  fs.chmodSync(filePath, 0o644);
}

async function main() {
  await waitForZitadel();
  const token = await waitForPat();
  const orgId = await getOrgId(token);
  await ensureRegistrationDisabled(token);
  await ensureAdminPasswordAccepted(token);
  const projectId = await ensureProject(token, orgId);
  await ensureRoles(token, projectId);
  const [applicationId, clientId] = await ensureFrontendApplication(token, projectId);
  const adminUserId = await ensureAdminUser(token, orgId);
  await assignAdminRole(token, orgId, projectId, adminUserId);
  await cleanupExtraAdminUsers(token, projectId, adminUserId);
  writeText(CLIENT_ID_FILE, clientId);
  writeText(
    BOOTSTRAP_STATE_FILE,
    JSON.stringify(
      {
        org_id: orgId,
        project_id: projectId,
        frontend_application_id: applicationId,
        frontend_client_id: clientId,
        frontend_redirect_uri: FRONTEND_URL,
      },
      null,
      2,
    ),
  );
  log("bootstrap complete");
}

main().catch((error) => {
  log(error.message);
  process.exit(1);
});

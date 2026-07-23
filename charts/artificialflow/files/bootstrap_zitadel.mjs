import http from "node:http";
import https from "node:https";
import fs from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";

class ZitadelError extends Error {
  constructor(status, body) {
    super(`ZITADEL request failed with status ${status}`);
    this.status = status;
    this.body = body;
  }
}

function env(name, fallback = "") {
  return (process.env[name] || fallback).trim();
}

function envWithLegacy(canonicalName, legacyName, fallback = "") {
  return env(canonicalName, env(legacyName, fallback));
}

function canonicalIdentity(value) {
  return String(value || "").trim().toLowerCase();
}

const ZITADEL_INTERNAL_URL = env("ZITADEL_INTERNAL_URL", "http://zitadel-api:8080").replace(/\/$/, "");
const ZITADEL_PUBLIC_URL = env("ZITADEL_PUBLIC_URL", "http://localhost:9180").replace(/\/$/, "");
const OWNER_PAT_FILE = env("ZITADEL_OWNER_PAT_FILE", "/zitadel/bootstrap/owner.pat");
const CLIENT_ID_FILE = envWithLegacy("ARTIFICIALFLOW_FRONTEND_CLIENT_ID_FILE", "FLOWGO_FRONTEND_CLIENT_ID_FILE", "/artificialflow/bootstrap/artificialflow-frontend-client-id");
const PROJECT_ID_FILE = envWithLegacy("ARTIFICIALFLOW_PROJECT_ID_FILE", "FLOWGO_PROJECT_ID_FILE", "/artificialflow/bootstrap/artificialflow-project-id");
const API_CLIENT_ID_FILE = envWithLegacy("ARTIFICIALFLOW_API_CLIENT_ID_FILE", "FLOWGO_API_CLIENT_ID_FILE", "/artificialflow/auth/artificialflow-api-client-id");
const API_CLIENT_SECRET_FILE = envWithLegacy("ARTIFICIALFLOW_API_CLIENT_SECRET_FILE", "FLOWGO_API_CLIENT_SECRET_FILE", "/artificialflow/auth/artificialflow-api-client-secret");
const API_CREDENTIAL_UID = Number(envWithLegacy("ARTIFICIALFLOW_API_CREDENTIAL_UID", "FLOWGO_API_CREDENTIAL_UID", "10001"));
const BOOTSTRAP_STATE_FILE = envWithLegacy("ARTIFICIALFLOW_ZITADEL_BOOTSTRAP_STATE_FILE", "FLOWGO_ZITADEL_BOOTSTRAP_STATE_FILE", "/artificialflow/bootstrap/artificialflow-zitadel.json");
const PROJECT_NAME = envWithLegacy("ARTIFICIALFLOW_PROJECT_NAME", "FLOWGO_PROJECT_NAME", "ArtificialFlow");
const FRONTEND_APP_NAME = envWithLegacy("ARTIFICIALFLOW_FRONTEND_APP_NAME", "FLOWGO_FRONTEND_APP_NAME", "ArtificialFlow Frontend");
const API_APP_NAME = envWithLegacy("ARTIFICIALFLOW_API_APP_NAME", "FLOWGO_API_APP_NAME", "ArtificialFlow API");
const LEGACY_PROJECT_NAMES = new Set(["FlowGo", ...envWithLegacy("ARTIFICIALFLOW_LEGACY_PROJECT_NAMES", "FLOWGO_LEGACY_PROJECT_NAMES").split(",")].map(canonicalIdentity).filter(Boolean));
const LEGACY_FRONTEND_APP_NAMES = new Set(["FlowGo Frontend", ...envWithLegacy("ARTIFICIALFLOW_LEGACY_FRONTEND_APP_NAMES", "FLOWGO_LEGACY_FRONTEND_APP_NAMES").split(",")].map(canonicalIdentity).filter(Boolean));
const LEGACY_API_APP_NAMES = new Set(["FlowGo API", ...envWithLegacy("ARTIFICIALFLOW_LEGACY_API_APP_NAMES", "FLOWGO_LEGACY_API_APP_NAMES").split(",")].map(canonicalIdentity).filter(Boolean));
const FRONTEND_URL = envWithLegacy("ARTIFICIALFLOW_FRONTEND_URL", "FLOWGO_FRONTEND_URL", "http://localhost:9100").replace(/\/$/, "");
const ACCESS_TOKEN_LIFETIME = envWithLegacy("ARTIFICIALFLOW_ZITADEL_ACCESS_TOKEN_LIFETIME", "FLOWGO_ZITADEL_ACCESS_TOKEN_LIFETIME", "900s");
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
  ["artificialflow admin", "ArtificialFlow Admin"],
  ["artificialflow modeler", "ArtificialFlow Modeler"],
  ["artificialflow client", "ArtificialFlow Client"],
];
const LEGACY_ROLE_MAPPINGS = new Map([
  ["flowgo admin", "artificialflow admin"],
  ["flowgo modeler", "artificialflow modeler"],
  ["flowgo client", "artificialflow client"],
]);

function canonicalRoleKey(roleKey) {
  const trimmed = String(roleKey || "").trim();
  const normalized = trimmed.toLowerCase();
  return LEGACY_ROLE_MAPPINGS.get(normalized) || ROLES.find(([role]) => role === normalized)?.[0] || trimmed;
}

function canonicalRoleKeys(roleKeys) {
  const seen = new Set();
  const canonical = [];
  for (const roleKey of roleKeys || []) {
    const role = canonicalRoleKey(roleKey);
    const key = role.toLowerCase();
    if (!role || seen.has(key)) continue;
    seen.add(key);
    canonical.push(role);
  }
  return canonical;
}

function sameStrings(left, right) {
  return left.length === right.length && left.every((value, index) => value === right[index]);
}

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

function paginationTotal(response) {
  const raw = response.pagination?.totalResult ?? response.details?.totalResult;
  if (raw === undefined || raw === null || raw === "") return null;
  const total = Number(raw);
  return Number.isSafeInteger(total) && total >= 0 ? total : null;
}

async function collectPaginated(fetchPage, resultKey, limit = 100) {
  const results = [];
  let offset = 0;
  while (true) {
    const response = await fetchPage({
      offset: String(offset),
      limit: String(limit),
    });
    const page = response[resultKey] || [];
    if (!Array.isArray(page)) {
      throw new Error(`ZITADEL pagination response field ${resultKey} is not an array`);
    }
    results.push(...page);
    offset += page.length;
    const total = paginationTotal(response);
    if (page.length === 0 || (total !== null && offset >= total)) return results;
    if (total === null && page.length < limit) return results;
  }
}

function listPaginated(token, requestPath, resultKey, limit = 100, envelopeField = "pagination") {
  return collectPaginated(
    (page) => connect(requestPath, token, { [envelopeField]: page }),
    resultKey,
    limit,
  );
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
  return listPaginated(token, "/zitadel.project.v2.ProjectService/ListProjects", "projects");
}

function isNotFound(error) {
  const body = JSON.stringify(error?.body || error).toLowerCase();
  return error?.status === 404 || body.includes("not_found") || body.includes("not found");
}

async function getProjectById(token, projectId) {
  if (!projectId) return null;
  try {
    const response = await connect("/zitadel.project.v2.ProjectService/GetProject", token, { projectId });
    return response.project || null;
  } catch (error) {
    if (isNotFound(error)) return null;
    throw error;
  }
}

function readBootstrapState() {
  const raw = readTextIfPresent(BOOTSTRAP_STATE_FILE);
  if (!raw) return {};
  try {
    return JSON.parse(raw);
  } catch (error) {
    throw new Error(`invalid legacy bootstrap state at ${BOOTSTRAP_STATE_FILE}: ${error.message}`);
  }
}

function selectProject(projects, state = {}) {
  const preferredId = state.project_id || state.projectId || readTextIfPresent(PROJECT_ID_FILE);
  if (preferredId) {
    const byId = projects.find((project) => project.projectId === preferredId);
    if (byId) return byId;
  }
  const canonicalName = canonicalIdentity(PROJECT_NAME);
  return projects.find((project) => canonicalIdentity(project.name) === canonicalName)
    || projects.find((project) => LEGACY_PROJECT_NAMES.has(canonicalIdentity(project.name)));
}

async function renameProject(token, project) {
  if (project.name === PROJECT_NAME) return;
  await connect("/zitadel.project.v2.ProjectService/UpdateProject", token, {
    projectId: project.projectId,
    name: PROJECT_NAME,
  });
  log(`renamed project ${project.name} to ${PROJECT_NAME} (${project.projectId})`);
}

async function ensureProject(token, orgId, state) {
  const preferredId = state.project_id || state.projectId || readTextIfPresent(PROJECT_ID_FILE);
  const existing = await getProjectById(token, preferredId)
    || selectProject(await listProjects(token), state);
  if (existing) {
    await renameProject(token, existing);
    log(`using existing project ${PROJECT_NAME} (${existing.projectId})`);
    return existing.projectId;
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

function durationSeconds(value) {
  const match = /^([1-9]\d*)(s|m|h)$/.exec(value);
  if (!match) {
    throw new Error("ARTIFICIALFLOW_ZITADEL_ACCESS_TOKEN_LIFETIME must be a positive duration using s, m, or h");
  }
  const multiplier = { s: 1, m: 60, h: 3600 }[match[2]];
  const seconds = Number(match[1]) * multiplier;
  if (!Number.isSafeInteger(seconds) || seconds < 60 || seconds > 86400) {
    throw new Error("ARTIFICIALFLOW_ZITADEL_ACCESS_TOKEN_LIFETIME must be between 60s and 24h");
  }
  return seconds;
}

async function ensureAccessTokenLifetime(token) {
  const desiredSeconds = durationSeconds(ACCESS_TOKEN_LIFETIME);
  const response = await requestJson("GET", "/admin/v1/settings/oidc", token, undefined, [200]);
  const settings = response.settings || {};
  const required = ["accessTokenLifetime", "idTokenLifetime", "refreshTokenIdleExpiration", "refreshTokenExpiration"];
  for (const field of required) {
    if (!settings[field]) {
      throw new Error(`ZITADEL OIDC settings response omitted ${field}; refusing a partial settings update`);
    }
  }
  if (durationSeconds(settings.accessTokenLifetime) === desiredSeconds) {
    log(`access-token lifetime is already ${ACCESS_TOKEN_LIFETIME}`);
    return;
  }
  const payload = {
    accessTokenLifetime: `${desiredSeconds}s`,
    idTokenLifetime: settings.idTokenLifetime,
    refreshTokenIdleExpiration: settings.refreshTokenIdleExpiration,
    refreshTokenExpiration: settings.refreshTokenExpiration,
  };
  const inherited = response.isDefault === true || settings.isDefault === true;
  const method = inherited ? "POST" : "PUT";
  try {
    await requestJson(method, "/admin/v1/settings/oidc", token, payload, [200]);
  } catch (error) {
    if (method === "POST" && isAlreadyExists(error)) {
      await requestJson("PUT", "/admin/v1/settings/oidc", token, payload, [200]);
    } else if (!isNotChanged(error)) {
      throw error;
    }
  }
  log(`configured access-token lifetime to ${ACCESS_TOKEN_LIFETIME}`);
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
        group: "ArtificialFlow",
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

async function migrateRoleAssignments(token, projectId) {
  for (const authorization of await listProjectAuthorizations(token)) {
    if (authorization.project?.id !== projectId || !authorization.id) continue;
    const current = (authorization.roles || []).map((role) => role.key).filter(Boolean);
    const canonical = canonicalRoleKeys(current);
    if (sameStrings(current, canonical)) continue;
    await connect("/zitadel.authorization.v2.AuthorizationService/UpdateAuthorization", token, {
      id: authorization.id,
      roleKeys: canonical,
    });
    log(`migrated role assignment ${authorization.id} without changing custom roles`);
  }
}

async function listApplications(token) {
  return listPaginated(token, "/zitadel.application.v2.ApplicationService/ListApplications", "applications");
}

async function getApplicationById(token, applicationId) {
  if (!applicationId) return null;
  try {
    const response = await connect("/zitadel.application.v2.ApplicationService/GetApplication", token, { applicationId });
    return response.application || null;
  } catch (error) {
    if (isNotFound(error)) return null;
    throw error;
  }
}

function selectApplication(applications, {
  projectId,
  applicationId,
  clientId,
  name,
  legacyNames,
  configurationKey,
}) {
  const candidates = applications.filter((app) => app.projectId === projectId && app[configurationKey]?.clientId);
  if (applicationId) {
    const byId = candidates.find((app) => app.applicationId === applicationId);
    if (byId) return byId;
  }
  if (clientId) {
    const byClientId = candidates.find((app) => app[configurationKey].clientId === clientId);
    if (byClientId) return byClientId;
  }
  const canonicalName = canonicalIdentity(name);
  return candidates.find((app) => canonicalIdentity(app.name) === canonicalName)
    || candidates.find((app) => legacyNames.has(canonicalIdentity(app.name)));
}

async function findApplication(token, options) {
  const saved = await getApplicationById(token, options.applicationId);
  if (saved) {
    const selected = selectApplication([saved], options);
    if (!selected) {
      throw new Error(`saved application ${options.applicationId} does not match the expected project and type`);
    }
    return selected;
  }
  return selectApplication(await listApplications(token), options);
}

async function renameApplication(token, projectId, application, name) {
  if (application.name === name) return;
  const configurationKey = application.oidcConfiguration
    ? "oidcConfiguration"
    : application.apiConfiguration
      ? "apiConfiguration"
      : "";
  if (!configurationKey) {
    throw new Error(`cannot rename unsupported application type ${application.applicationId}`);
  }
  const payload = {
    projectId,
    applicationId: application.applicationId,
    name,
    [configurationKey]: {},
  };
  await connect("/zitadel.application.v2.ApplicationService/UpdateApplication", token, payload);
  log(`renamed application ${application.name} to ${name} (${application.applicationId})`);
}

async function ensureFrontendApplication(token, projectId, state) {
  const existing = await findApplication(token, {
    projectId,
    applicationId: state.frontend_application_id || state.frontendApplicationId,
    clientId: state.frontend_client_id || state.frontendClientId || readTextIfPresent(CLIENT_ID_FILE),
    name: FRONTEND_APP_NAME,
    legacyNames: LEGACY_FRONTEND_APP_NAMES,
    configurationKey: "oidcConfiguration",
  });
  if (existing) {
    await renameApplication(token, projectId, existing, FRONTEND_APP_NAME);
    const clientId = existing.oidcConfiguration.clientId;
    log(`using existing frontend app ${FRONTEND_APP_NAME} with client ID ${clientId}`);
    return [existing.applicationId, clientId];
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

function readTextIfPresent(filePath) {
  try {
    return fs.readFileSync(filePath, "utf8").trim();
  } catch {
    return "";
  }
}

async function ensureAPIApplication(token, projectId, state) {
  const storedClientId = readTextIfPresent(API_CLIENT_ID_FILE) || state.api_client_id || state.apiClientId;
  const existing = await findApplication(token, {
    projectId,
    applicationId: state.api_application_id || state.apiApplicationId,
    clientId: storedClientId,
    name: API_APP_NAME,
    legacyNames: LEGACY_API_APP_NAMES,
    configurationKey: "apiConfiguration",
  });
  if (existing) {
    await renameApplication(token, projectId, existing, API_APP_NAME);
    const clientId = existing.apiConfiguration.clientId;
    const clientSecret = readTextIfPresent(API_CLIENT_SECRET_FILE);
    if (!clientSecret || (storedClientId && storedClientId !== clientId)) {
      throw new Error(`existing API application ${existing.applicationId} was found, but its preserved client secret is unavailable; refusing to rotate credentials`);
    }
    log(`using existing API application ${API_APP_NAME} with client ID ${clientId}`);
    return [existing.applicationId, clientId, clientSecret];
  }

  const response = await connect("/zitadel.application.v2.ApplicationService/CreateApplication", token, {
    projectId,
    name: API_APP_NAME,
    apiConfiguration: {
      authMethodType: "API_AUTH_METHOD_TYPE_BASIC",
    },
  });
  const clientId = response.apiConfiguration?.clientId;
  const clientSecret = response.apiConfiguration?.clientSecret;
  if (!clientId || !clientSecret) {
    throw new Error("ZITADEL did not return API introspection credentials");
  }
  log(`created API application ${API_APP_NAME} with client ID ${clientId}`);
  return [response.applicationId, clientId, clientSecret];
}

async function listUsers(token) {
  return listPaginated(token, "/zitadel.user.v2.UserService/ListUsers", "result", 500, "query");
}

async function getUserById(token, userId) {
  if (!userId) return null;
  try {
    const response = await connect("/zitadel.user.v2.UserService/GetUserByID", token, { userId });
    return response.user || null;
  } catch (error) {
    if (isNotFound(error)) return null;
    throw error;
  }
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

async function listProjectAuthorizations(token) {
  return listPaginated(
    token,
    "/zitadel.authorization.v2.AuthorizationService/ListAuthorizations",
    "authorizations",
    500,
  );
}

function authorizationHasAdminRole(authorization, projectId) {
  return authorization.project?.id === projectId
    && (authorization.roles || []).some((role) => canonicalRoleKey(role.key) === "artificialflow admin");
}

async function findAdminUser(token, projectId, state) {
  const preferredUserId = state.admin_user_id || state.adminUserId;
  const preferred = await getUserById(token, preferredUserId);
  if (preferred) {
    if (!preferred.human) {
      throw new Error(`saved admin user ${preferredUserId} is not a human user`);
    }
    return { user: preferred, updateProfile: isSolutionAdminUser(preferred) };
  }

  const users = await listUsers(token);

  const configured = users.find(isTargetAdminLoginUser) || users.find(isSolutionAdminUser);
  if (configured) return { user: configured, updateProfile: true };

  const legacy = users.find(isLegacySolutionAdminUser);
  if (legacy) return { user: legacy, updateProfile: true };

  const adminUserIds = new Set(
    (await listProjectAuthorizations(token))
      .filter((authorization) => authorizationHasAdminRole(authorization, projectId))
      .map((authorization) => authorization.user?.id)
      .filter(Boolean),
  );
  const assignedAdmin = users.find((user) => user.human && adminUserIds.has(user.userId));
  return assignedAdmin ? { user: assignedAdmin, updateProfile: false } : null;
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

async function ensureAdminUser(token, orgId, projectId, state) {
  const existing = await findAdminUser(token, projectId, state);
  if (existing) {
    if (existing.updateProfile) await updateAdminUser(token, existing.user);
    log(`using existing admin user ${existing.user.preferredLoginName || existing.user.username || existing.user.userId}`);
    return existing.user.userId;
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

async function assignAdminRole(token, orgId, projectId, userId) {
  if (!userId) {
    log(`admin user not found, skipping role assignment: ${ADMIN_LOGIN_NAME}`);
    return;
  }
  const existing = (await listProjectAuthorizations(token)).find(
    (authorization) => authorization.project?.id === projectId && authorization.user?.id === userId,
  );
  if (existing) {
    const current = (existing.roles || []).map((role) => role.key).filter(Boolean);
    const roleKeys = canonicalRoleKeys([...current, "artificialflow admin"]);
    if (!sameStrings(current, roleKeys)) {
      await connect("/zitadel.authorization.v2.AuthorizationService/UpdateAuthorization", token, {
        id: existing.id,
        roleKeys,
      });
    }
    if (existing.state && existing.state !== "STATE_ACTIVE") {
      await connect("/zitadel.authorization.v2.AuthorizationService/ActivateAuthorization", token, { id: existing.id });
    }
    log(`verified artificialflow admin for ${ADMIN_LOGIN_NAME}`);
    return;
  }
  await connect("/zitadel.authorization.v2.AuthorizationService/CreateAuthorization", token, {
    userId,
    projectId,
    organizationId: orgId,
    roleKeys: ["artificialflow admin"],
  });
  log(`assigned artificialflow admin to ${ADMIN_LOGIN_NAME}`);
}

function writeText(filePath, value, mode = 0o644, ownerUid = undefined) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  const tempPath = `${filePath}.tmp`;
  fs.writeFileSync(tempPath, `${value}\n`, { encoding: "utf8", mode });
  fs.renameSync(tempPath, filePath);
  fs.chmodSync(filePath, mode);
  if (ownerUid !== undefined && Number.isInteger(ownerUid) && ownerUid >= 0) {
    fs.chownSync(filePath, ownerUid, ownerUid);
  }
}

async function main() {
  await waitForZitadel();
  const token = await waitForPat();
  const bootstrapState = readBootstrapState();
  const orgId = await getOrgId(token);
  await ensureRegistrationDisabled(token);
  await ensureAdminPasswordAccepted(token);
  await ensureAccessTokenLifetime(token);
  const projectId = await ensureProject(token, orgId, bootstrapState);
  await ensureRoles(token, projectId);
  await migrateRoleAssignments(token, projectId);
  const [applicationId, clientId] = await ensureFrontendApplication(token, projectId, bootstrapState);
  const [apiApplicationId, apiClientId, apiClientSecret] = await ensureAPIApplication(token, projectId, bootstrapState);
  const adminUserId = await ensureAdminUser(token, orgId, projectId, bootstrapState);
  await assignAdminRole(token, orgId, projectId, adminUserId);
  writeText(CLIENT_ID_FILE, clientId);
  writeText(PROJECT_ID_FILE, projectId);
  writeText(API_CLIENT_ID_FILE, apiClientId, 0o600, API_CREDENTIAL_UID);
  writeText(API_CLIENT_SECRET_FILE, apiClientSecret, 0o600, API_CREDENTIAL_UID);
  writeText(
    BOOTSTRAP_STATE_FILE,
    JSON.stringify(
      {
        ...bootstrapState,
        org_id: orgId,
        project_id: projectId,
        frontend_application_id: applicationId,
        frontend_client_id: clientId,
        frontend_redirect_uri: FRONTEND_URL,
        api_application_id: apiApplicationId,
        api_client_id: apiClientId,
        admin_user_id: adminUserId,
      },
      null,
      2,
    ),
  );
  log("bootstrap complete");
}

export {
  canonicalRoleKey,
  canonicalRoleKeys,
  collectPaginated,
  envWithLegacy,
  listUsers,
  selectApplication,
  selectProject,
};

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  main().catch((error) => {
    log(error.message);
    process.exit(1);
  });
}

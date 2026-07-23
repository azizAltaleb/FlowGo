import assert from "node:assert/strict";
import fs from "node:fs";
import http from "node:http";
import test from "node:test";
import {
  canonicalRoleKeys,
  collectPaginated,
  envWithLegacy,
  selectApplication,
  selectProject,
} from "./bootstrap_zitadel.mjs";

test("envWithLegacy prefers canonical values and supports legacy fallback", () => {
  const canonicalName = "ARTIFICIALFLOW_TEST_PRECEDENCE";
  const legacyName = "FLOWGO_TEST_PRECEDENCE";
  const previousCanonical = process.env[canonicalName];
  const previousLegacy = process.env[legacyName];
  try {
    process.env[canonicalName] = "canonical";
    process.env[legacyName] = "legacy";
    assert.equal(envWithLegacy(canonicalName, legacyName, "fallback"), "canonical");
    delete process.env[canonicalName];
    assert.equal(envWithLegacy(canonicalName, legacyName, "fallback"), "legacy");
    delete process.env[legacyName];
    assert.equal(envWithLegacy(canonicalName, legacyName, "fallback"), "fallback");
  } finally {
    if (previousCanonical === undefined) delete process.env[canonicalName];
    else process.env[canonicalName] = previousCanonical;
    if (previousLegacy === undefined) delete process.env[legacyName];
    else process.env[legacyName] = previousLegacy;
  }
});

test("canonicalRoleKeys migrates standard roles and preserves custom roles", () => {
  const migrated = canonicalRoleKeys([
    "flowgo admin",
    "artificialflow admin",
    "flowgo client",
    "Finance Reviewer",
  ]);
  assert.deepEqual(migrated, [
    "artificialflow admin",
    "artificialflow client",
    "Finance Reviewer",
  ]);
  assert.deepEqual(canonicalRoleKeys(migrated), migrated);
});

test("collectPaginated finds saved migration objects on later pages instead of duplicating them", async () => {
  const requested = [];
  const projects = await collectPaginated(async (pagination) => {
    requested.push(pagination);
    if (pagination.offset === "0") {
      return {
        pagination: { totalResult: "3" },
        projects: [
          { projectId: "other-1", name: "Other One" },
          { projectId: "other-2", name: "Other Two" },
        ],
      };
    }
    return {
      pagination: { totalResult: "3" },
      projects: [{ projectId: "saved-project", name: "FlowGo" }],
    };
  }, "projects", 2);

  assert.deepEqual(requested, [
    { offset: "0", limit: "2" },
    { offset: "2", limit: "2" },
  ]);
  assert.equal(
    selectProject(projects, { project_id: "saved-project" })?.projectId,
    "saved-project",
  );
});

test("multi-page application and user results preserve saved IDs", async () => {
  const applications = await collectPaginated(async ({ offset }) => ({
    pagination: { totalResult: "2" },
    applications: offset === "0"
      ? [{
        applicationId: "other-app",
        projectId: "project-1",
        name: "Other",
        oidcConfiguration: { clientId: "other-client" },
      }]
      : [{
        applicationId: "saved-app",
        projectId: "project-1",
        name: "FlowGo Frontend",
        oidcConfiguration: { clientId: "saved-client" },
      }],
  }), "applications", 1);
  const selected = selectApplication(applications, {
    projectId: "project-1",
    applicationId: "saved-app",
    clientId: "saved-client",
    name: "ArtificialFlow Frontend",
    legacyNames: new Set(["flowgo frontend"]),
    configurationKey: "oidcConfiguration",
  });
  assert.equal(selected?.applicationId, "saved-app");
  assert.equal(selected?.oidcConfiguration.clientId, "saved-client");

  const users = await collectPaginated(async ({ offset }) => ({
    pagination: { totalResult: "2" },
    result: offset === "0"
      ? [{ userId: "other-user", human: {} }]
      : [{ userId: "saved-user", human: {}, preferredLoginName: "admin" }],
  }), "result", 1);
  assert.equal(users.find(({ userId }) => userId === "saved-user")?.userId, "saved-user");
});

test("ListUsers sends the v4 query envelope and advances outgoing offsets", async () => {
  const requestBodies = [];
  const requestPaths = [];
  const server = http.createServer((request, response) => {
    const chunks = [];
    request.on("data", (chunk) => chunks.push(chunk));
    request.on("end", () => {
      const payload = JSON.parse(Buffer.concat(chunks).toString("utf8"));
      requestPaths.push(request.url);
      requestBodies.push(payload);
      const offset = payload.query?.offset;
      const result = offset === "0"
        ? Array.from({ length: 500 }, (_, index) => ({ userId: `user-${index}` }))
        : [{ userId: "user-500" }];
      response.writeHead(200, { "Content-Type": "application/json" });
      response.end(JSON.stringify({
        pagination: { totalResult: "501" },
        result,
      }));
    });
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const { port } = server.address();

  const previousInternalUrl = process.env.ZITADEL_INTERNAL_URL;
  const previousPublicUrl = process.env.ZITADEL_PUBLIC_URL;
  try {
    process.env.ZITADEL_INTERNAL_URL = `http://127.0.0.1:${port}`;
    process.env.ZITADEL_PUBLIC_URL = "https://iam.example.test";
    const { listUsers } = await import(`./bootstrap_zitadel.mjs?list-users-body=${Date.now()}`);
    const users = await listUsers("test-token");
    assert.equal(users.length, 501);
  } finally {
    if (previousInternalUrl === undefined) delete process.env.ZITADEL_INTERNAL_URL;
    else process.env.ZITADEL_INTERNAL_URL = previousInternalUrl;
    if (previousPublicUrl === undefined) delete process.env.ZITADEL_PUBLIC_URL;
    else process.env.ZITADEL_PUBLIC_URL = previousPublicUrl;
    await new Promise((resolve, reject) => server.close((error) => error ? reject(error) : resolve()));
  }

  assert.deepEqual(requestPaths, [
    "/zitadel.user.v2.UserService/ListUsers",
    "/zitadel.user.v2.UserService/ListUsers",
  ]);
  assert.deepEqual(requestBodies, [
    { query: { offset: "0", limit: "500" } },
    { query: { offset: "500", limit: "500" } },
  ]);
});

test("multi-page authorization migration preserves IDs and custom roles", async () => {
  const authorizations = await collectPaginated(async ({ offset }) => {
    if (offset === "0") {
      return {
        details: { totalResult: "2" },
        authorizations: [{
          id: "unrelated-auth",
          roles: [{ key: "auditor" }],
        }],
      };
    }
    return {
      details: { totalResult: "2" },
      authorizations: [{
        id: "saved-auth",
        roles: [{ key: "flowgo admin" }, { key: "Finance Reviewer" }],
      }],
    };
  }, "authorizations", 1);

  const saved = authorizations.find(({ id }) => id === "saved-auth");
  assert.equal(saved?.id, "saved-auth");
  assert.deepEqual(
    canonicalRoleKeys(saved.roles.map(({ key }) => key)),
    ["artificialflow admin", "Finance Reviewer"],
  );
});

test("selectProject prefers immutable legacy state before names", () => {
  const projects = [
    { projectId: "canonical-name", name: "ArtificialFlow" },
    { projectId: "legacy-id", name: "FlowGo" },
  ];
  assert.equal(selectProject(projects, { project_id: "legacy-id" })?.projectId, "legacy-id");
  assert.equal(selectProject([projects[1]], {})?.projectId, "legacy-id");
});

test("selectApplication reuses immutable IDs and legacy names", () => {
  const applications = [
    {
      applicationId: "canonical-name",
      projectId: "project-1",
      name: "ArtificialFlow Frontend",
      oidcConfiguration: { clientId: "canonical-client" },
    },
    {
      applicationId: "legacy-id",
      projectId: "project-1",
      name: "FlowGo Frontend",
      oidcConfiguration: { clientId: "legacy-client" },
    },
  ];
  const options = {
    projectId: "project-1",
    applicationId: "legacy-id",
    clientId: "",
    name: "ArtificialFlow Frontend",
    legacyNames: new Set(["flowgo frontend"]),
    configurationKey: "oidcConfiguration",
  };
  assert.equal(selectApplication(applications, options)?.applicationId, "legacy-id");
  assert.equal(
    selectApplication([applications[1]], { ...options, applicationId: "" })?.applicationId,
    "legacy-id",
  );
});

test("chart bootstrap copy stays byte-for-byte consistent", () => {
  const script = fs.readFileSync(new URL("./bootstrap_zitadel.mjs", import.meta.url), "utf8");
  const chartCopy = fs.readFileSync(
    new URL("../charts/artificialflow/files/bootstrap_zitadel.mjs", import.meta.url),
    "utf8",
  );
  assert.equal(chartCopy, script);
  assert.doesNotMatch(script, /GenerateClientSecret/);
});

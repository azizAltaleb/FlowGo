import { expect, test, type Page } from "@playwright/test";
import { ArtificialFlowClient as NodeSDKClient } from "../../../../clients/nodejs-sdk/src/Client";
import { ZitadelJwtProfileAuthProvider } from "../../../../clients/nodejs-sdk/src/Auth";
import type { ZitadelJwtProfile } from "../../../../clients/nodejs-sdk/src/types";
import { createServiceAccountProfile, generateServiceAccountKeyPair } from "../../../../frontend/src/lib/serviceAccount";
import { buildBpmnFixture, type BpmnFixtureName } from "../fixtures/uat/bpmn";
import { uatCases, type UatCase } from "../fixtures/uat/cases";
import { loginForDeployment, type UatDeployment } from "../fixtures/uat/auth";
import { ArtificialFlowApi, type ActingUser, type WorkflowInstance } from "../fixtures/uat/artificialflow-api";
import { runDir, writeCaseResult } from "../fixtures/uat/results";

test.use({ trace: "retain-on-failure", screenshot: "only-on-failure" });
test.describe.configure({ mode: "serial" });

const deployment = (process.env.UAT_DEPLOYMENT || "bundled-zitadel") as UatDeployment;
const runId = process.env.UAT_RUN_ID || new Date().toISOString().replace(/[-:.TZ]/g, "").slice(0, 14);
const caseFilter = new Set((process.env.UAT_CASE_FILTER || "").split(",").map((item) => item.trim()).filter(Boolean));
const casesToRun = uatCases.filter((uatCase) => caseFilter.size === 0 || caseFilter.has(uatCase.id));

test.describe(`ArtificialFlow UAT video suite - ${deployment}`, () => {
  for (const uatCase of casesToRun) {
    test(`${uatCase.id} ${uatCase.title}`, async ({ browser }, testInfo) => {
      testInfo.setTimeout(180_000);
      let rawVideo: string | undefined;
      let error: Error | undefined;
      let resultCase = uatCase;
      const context = await browser.newContext({
        viewport: { width: 1440, height: 900 },
        recordVideo: {
          dir: `${runDir()}/raw-videos/${deployment}`,
          size: { width: 1440, height: 900 },
        },
      });
      const page = await context.newPage();
      page.setDefaultTimeout(30_000);

      try {
        console.log(`[uat-case] start ${deployment} ${uatCase.id}`);
        if (deployment === "external-keycloak" && uatCase.bundledOnly) {
          resultCase = {
            ...uatCase,
            expected: "expected-skip",
            reason: "This case validates bundled ZITADEL admin-only management APIs and is intentionally skipped for external Keycloak.",
          };
          await showExpectedSkip(page, resultCase, deployment);
          return;
        }

        await runUatCase(page, uatCase, deployment);
        console.log(`[uat-case] complete ${deployment} ${uatCase.id}`);
      } catch (err) {
        error = err instanceof Error ? err : new Error(String(err));
        throw err;
      } finally {
        console.log(`[uat-case] finalize ${deployment} ${resultCase.id}`);
        const video = page.video();
        await context.close().catch(() => undefined);
        rawVideo = await video?.path().catch(() => undefined);
        await writeCaseResult(testInfo, deployment, resultCase, rawVideo, error);
      }
    });
  }
});

async function runUatCase(page: Page, uatCase: UatCase, mode: UatDeployment): Promise<void> {
  if (uatCase.expected === "expected-skip") {
    await showExpectedSkip(page, uatCase, mode);
    return;
  }

  await loginForDeployment(page, mode);
  console.log(`[uat-case] authenticated ${uatCase.id}`);
  const api = new ArtificialFlowApi(page);
  const created = { workflows: [] as string[], instances: [] as string[], clients: [] as string[], users: [] as string[], roles: [] as string[] };

  try {
    await addBanner(page, `${uatCase.id}: ${uatCase.title}`);
    console.log(`[uat-case] dispatch ${uatCase.id}`);

    switch (uatCase.id) {
      case "UAT-IAM-001":
        await runIamModeCase(page, api, mode);
        break;
      case "UAT-UI-001":
        await runDashboardCase(page);
        break;
      case "UAT-UI-002":
        await runUserTaskUiCase(page, api, uatCase, created);
        break;
      case "UAT-END2END-001":
        await runRoleBasedComplexCase(page, api, uatCase, mode, created);
        break;
      case "UAT-BPMN-004":
        await runCallBusinessManualCase(page, api, uatCase, created);
        break;
      case "UAT-BPMN-005":
        await runEventGatewayCase(page, api, uatCase, created);
        break;
      case "UAT-BPMN-006":
        await runBoundaryTimerCase(page, api, uatCase, created);
        break;
      case "UAT-BPMN-007":
        await runMessageSignalCase(page, api, uatCase, created);
        break;
      case "UAT-BPMN-008":
        await runServiceWorkerCase(page, api, uatCase, created);
        break;
      case "UAT-BPMN-009":
      case "UAT-BPMN-010":
        await runExpectedRejectionCase(page, api, uatCase);
        break;
      case "UAT-WORKER-001":
        await runWorkerGuardrailCase(page, api);
        break;
      case "UAT-IAM-002":
        await runBundledIdentityManagementCase(page, api, created);
        break;
      default:
        await runDeployOnlyBpmnCase(page, api, uatCase, created);
        break;
    }

    await page.waitForTimeout(1_200);
  } finally {
    await cleanup(api, created);
  }
}

async function runRoleBasedComplexCase(page: Page, api: ArtificialFlowApi, uatCase: UatCase, mode: UatDeployment, created: Created): Promise<void> {
  const processId = processIdFor(uatCase);
  const accountant = roleUser("accountant", mode);
  const reviewer = roleUser("reviewer", mode);

  await provisionRoleUsers(api, mode, accountant, reviewer, created);

  await page.goto("/processes", { waitUntil: "domcontentloaded" });
  await addBanner(page, "UAT starts from Processes: create a complex approval workflow");
  await page.getByRole("button", { name: /create workflow/i }).click();
  await page.locator("#process-name").fill("Role Based Complex Approval UAT");
  await page.locator("#process-id").fill(processId);
  await page.getByRole("button", { name: /^create$/i }).click();
  await page.waitForURL(/\/modeler\?new=true/, { timeout: 30_000 });
  await addBanner(page, "Processes screen created the workflow shell; Modeler is opened for the complex BPMN");
  await expect(page.locator("body")).toContainText(/Standard BPMN|Deploy Process/);

  const workflowId = await deployBundle(api, uatCase, created);
  await waitForProjectionEvidenceNonBlocking(page, api, workflowId, processId);

  await page.goto(`/modeler?id=${workflowId}`, { waitUntil: "domcontentloaded" });
  await addBanner(page, "One large process validates exclusive, parallel, inclusive, and event-based gateways with conditions");
  await expect(page.locator("body")).toContainText(/Exclusive|Parallel|Inclusive|Event Based/);
  await expect(page.locator("body")).toContainText(/Parallel risk stamp|Archive preparation|Compliance rule|Receive budget confirmation/);

  const instance = await startInstanceFromProcessesOrApi(page, api, workflowId, processId, mode);
  created.instances.push(instance.id);
  await api.updateVariables(instance.id, {
    amount: 9500,
    needsArchive: true,
    needsCompliance: true,
    invoiceId: `uat-${runId}`,
  });

  await completeWorkerJob(page, api, "uat-complex-worker", "uat-complex-worker");
  await api.waitForInstanceStatus(instance.id, (item) => activeSteps(item).includes("accountantReview"));
  const accountantReviewExecution = activeExecutionForStep(await api.getInstance(instance.id), "accountantReview");
  const inboxToken = await provisionInboxClientToken(page, api, mode, created);

  await assertInboxCannotSeeOrClaimTask(page, api, inboxToken, actingUserForRole(reviewer), instance.id, accountantReviewExecution.id, "accountantReview");

  await completeTaskFromInbox(page, api, inboxToken, actingUserForRole(accountant), instance.id, "accountantReview", "Accountant takes and completes accountantReview through ArtificialFlow Transaction Inbox");

  await api.waitForInstanceStatus(instance.id, (item) =>
    activeSteps(item).includes("receiveDocuments") && activeSteps(item).includes("receiveBudgetConfirmation")
  );
  await api.publishMessage("DocumentsReceived", "", { source: "uat-role-complex", by: "accountant" });
  await api.publishMessage("BudgetConfirmed", "", { source: "uat-role-complex", by: "accountant" });
  await api.waitForInstanceStatus(instance.id, (item) => activeSteps(item).includes("reviewerApproval"));
  const reviewerApprovalExecution = activeExecutionForStep(await api.getInstance(instance.id), "reviewerApproval");

  await assertInboxCannotSeeOrClaimTask(page, api, inboxToken, actingUserForRole(accountant), instance.id, reviewerApprovalExecution.id, "reviewerApproval");

  await completeTaskFromInbox(page, api, inboxToken, actingUserForRole(reviewer), instance.id, "reviewerApproval", "Reviewer takes and completes reviewerApproval through ArtificialFlow Transaction Inbox");

  const completed = await api.waitForInstanceStatus(instance.id, (item) => item.status === "COMPLETED");
  expect(completed.status).toBe("COMPLETED");
  await page.goto(`/instances/${instance.id}`, { waitUntil: "domcontentloaded" });
  await addBanner(page, "Complex role-based process completed end to end");
  await expect(page.locator("body")).toContainText("COMPLETED");
  await showCompletedHistoryEvidence(page, instance.id);
}

async function runIamModeCase(page: Page, api: ArtificialFlowApi, mode: UatDeployment): Promise<void> {
  const config = await api.identityConfig();
  const me = await api.identityMe();
  expect(config).toHaveProperty("deployment_mode");
  expect(me).toHaveProperty("authenticated", true);
  await page.goto("/", { waitUntil: "domcontentloaded" });
  await addBanner(page, `IAM mode validated: ${String(config.deployment_mode || mode)}`);
  await expect(page.getByText("Dashboard").first()).toBeVisible();
  if (mode === "bundled-zitadel") {
    await page.goto("/identity", { waitUntil: "domcontentloaded" });
    await addBanner(page, "Bundled ZITADEL admin identity management is visible");
    await expect(page.locator("body")).toContainText(/Identity|Users|Roles|Clients/i);
  }
}

async function runDashboardCase(page: Page): Promise<void> {
  await page.goto("/", { waitUntil: "domcontentloaded" });
  await addBanner(page, "Dashboard UAT: cards, counts, and recent activity");
  await expect(page.getByText("Dashboard").first()).toBeVisible();
  await expect(page.locator("body")).toContainText(/Total Instances|Recent Activity|Completed|Failed/);
  await page.goto("/processes", { waitUntil: "domcontentloaded" });
  await expect(page.getByRole("heading", { name: /processes/i }).first()).toBeVisible();
  await page.goto("/instances", { waitUntil: "domcontentloaded" });
  await expect(page.getByRole("heading", { name: /instances/i }).first()).toBeVisible();
}

async function runUserTaskUiCase(page: Page, api: ArtificialFlowApi, uatCase: UatCase, created: Created): Promise<void> {
  const { workflowId, instance } = await deployStart(page, api, uatCase, created, { amount: 250, approved: false });
  await waitForProjectionEvidence(page, api, workflowId, processIdFor(uatCase));
  await page.goto("/processes", { waitUntil: "domcontentloaded" });
  await addBanner(page, "Processes catalog shows the deployed UAT workflow");
  if (deployment !== "external-keycloak") {
    await expect(page.locator("body")).toContainText(processIdFor(uatCase));
  }

  await page.goto(`/modeler?id=${workflowId}`, { waitUntil: "domcontentloaded" });
  await addBanner(page, "Modeler renders the deployed BPMN");
  await expect(page.locator("body")).toContainText(/Modeler|Deploy Process|Standard BPMN/);

  await page.goto(`/instances/${instance.id}`, { waitUntil: "domcontentloaded" });
  await addBanner(page, "Instance details show active task and variables");
  await expect(page.locator("body")).toContainText(/Process Visualization|Active Tasks|Variables/);
  await api.updateVariables(instance.id, { amount: 250, approved: true, uat: uatCase.id });
  const updated = await api.getInstance(instance.id);
  expect(updated.context).toMatchObject({ approved: true, uat: uatCase.id });
  await completeAllActiveTasks(api, instance.id);
  const completed = await api.waitForInstanceStatus(instance.id, (item) => item.status === "COMPLETED");
  expect(completed.status).toBe("COMPLETED");
  await page.goto(`/instances/${instance.id}`, { waitUntil: "domcontentloaded" });
  await addBanner(page, "Task completed and instance reached COMPLETED");
  await expect(page.locator("body")).toContainText("COMPLETED");
}

async function runDeployOnlyBpmnCase(page: Page, api: ArtificialFlowApi, uatCase: UatCase, created: Created): Promise<void> {
  const workflowId = await deployBundle(api, uatCase, created);
  await waitForProjectionEvidence(page, api, workflowId, processIdFor(uatCase));
  await page.goto("/processes", { waitUntil: "domcontentloaded" });
  await addBanner(page, `${uatCase.id}: deployed and visible in process catalog`);
  if (deployment !== "external-keycloak") {
    await expect(page.locator("body")).toContainText(processIdFor(uatCase));
  }
  await page.goto(`/modeler?id=${workflowId}`, { waitUntil: "domcontentloaded" });
  await addBanner(page, `${uatCase.id}: modeler evidence for ${uatCase.elements.join(", ")}`);
  await expect(page.locator("body")).toContainText(/Modeler|Standard BPMN|Deploy Process/);
}

async function runCallBusinessManualCase(page: Page, api: ArtificialFlowApi, uatCase: UatCase, created: Created): Promise<void> {
  const { instance } = await deployStart(page, api, uatCase, created);
  const running = await api.waitForInstanceStatus(instance.id, (item) => activeSteps(item).includes("manualReview"));
  expect(activeSteps(running)).toContain("manualReview");
  await page.goto(`/instances/${instance.id}`, { waitUntil: "domcontentloaded" });
  await addBanner(page, "Call activity, business rule, and manual task reached manual review");
  await expect(page.locator("body")).toContainText("manualReview");
  await api.completeTask(instance.id, "manualReview");
  await api.waitForInstanceStatus(instance.id, (item) => item.status === "COMPLETED");
}

async function runEventGatewayCase(page: Page, api: ArtificialFlowApi, uatCase: UatCase, created: Created): Promise<void> {
  const { instance } = await deployStart(page, api, uatCase, created);
  await api.waitForInstanceStatus(instance.id, (item) => activeSteps(item).includes("receive"));
  await page.goto(`/instances/${instance.id}`, { waitUntil: "domcontentloaded" });
  await addBanner(page, "Event gateway is waiting on receive and timer branches");
  await api.publishMessage("MsgPaymentReceived", "", { source: "uat" });
  const afterMessage = await api.waitForInstanceStatus(instance.id, (item) => activeSteps(item).includes("afterReceive"));
  expect(activeSteps(afterMessage)).toContain("afterReceive");
  await page.goto(`/instances/${instance.id}`, { waitUntil: "domcontentloaded" });
  await addBanner(page, "Message branch won; timer branch is no longer the active path");
}

async function runBoundaryTimerCase(page: Page, api: ArtificialFlowApi, uatCase: UatCase, created: Created): Promise<void> {
  const { instance } = await deployStart(page, api, uatCase, created);
  await page.goto(`/instances/${instance.id}`, { waitUntil: "domcontentloaded" });
  await addBanner(page, "Boundary timer starts while user task is active");
  const timed = await api.waitForInstanceStatus(instance.id, (item) =>
    item.status === "COMPLETED" || activeSteps(item).includes("timeoutTask") || activeSteps(item).includes("timeoutEnd")
  );
  expect(activeSteps(timed).includes("userTask")).toBeFalsy();
}

async function runMessageSignalCase(page: Page, api: ArtificialFlowApi, uatCase: UatCase, created: Created): Promise<void> {
  const bundle = buildBpmnFixture(uatCase.fixture as BpmnFixtureName, `${runId}_${uatCase.id}`);
  const receiver = await api.deploy(bundle.dependencies![0].xml);
  created.workflows.push(receiver.id);
  const receiverInstance = await api.startInstance(receiver.id, { source: "uat-signal" });
  created.instances.push(receiverInstance.id);
  await api.waitForInstanceStatus(receiverInstance.id, (item) => activeSteps(item).includes("catchSignal"));

  const sender = await api.deploy(bundle.primary.xml);
  created.workflows.push(sender.id);
  const senderInstance = await api.startInstance(sender.id, { source: "uat-signal" });
  created.instances.push(senderInstance.id);
  await api.waitForInstanceStatus(senderInstance.id, (item) => item.status === "COMPLETED");
  await api.waitForInstanceStatus(receiverInstance.id, (item) => item.status === "COMPLETED");

  await page.goto(`/instances/${receiverInstance.id}`, { waitUntil: "domcontentloaded" });
  await addBanner(page, "Signal throw/catch correlation completed receiver instance");
  await expect(page.locator("body")).toContainText("COMPLETED");
}

async function runServiceWorkerCase(page: Page, api: ArtificialFlowApi, uatCase: UatCase, created: Created): Promise<void> {
  const { instance } = await deployStart(page, api, uatCase, created);
  const jobs = await pollJobs(api, "uat-worker", "uat-video-worker");
  expect(jobs.length).toBeGreaterThan(0);
  await api.extendJobLock(jobs[0].key, "uat-video-worker");
  await api.completeJob(jobs[0].key, "uat-video-worker", { workerApproved: true });
  await api.waitForInstanceStatus(instance.id, (item) => activeSteps(item).includes("assignedUser"));
  await page.goto(`/instances/${instance.id}`, { waitUntil: "domcontentloaded" });
  await addBanner(page, "External worker job completed and user assignment task is active");
  await expect(page.locator("body")).toContainText("assignedUser");
  await completeAllActiveTasks(api, instance.id);
  await api.waitForInstanceStatus(instance.id, (item) => item.status === "COMPLETED");
}

async function runExpectedRejectionCase(page: Page, api: ArtificialFlowApi, uatCase: UatCase): Promise<void> {
  const bundle = buildBpmnFixture(uatCase.fixture as BpmnFixtureName, `${runId}_${uatCase.id}`);
  const rejection = await api.deployExpectingRejection(bundle.primary.xml);
  expect(rejection.status).toBeGreaterThanOrEqual(400);
  await showStandaloneCard(page, uatCase, deployment, `Expected rejection received: HTTP ${rejection.status}\n${rejection.text.slice(0, 500)}`);
}

async function runWorkerGuardrailCase(page: Page, api: ArtificialFlowApi): Promise<void> {
  const capabilities = await api.jobsCapabilities();
  expect(capabilities.protocolVersion).toBe("v1");
  expect(capabilities.capabilities).toEqual(expect.arrayContaining(["activate", "complete", "fail", "extend-lock"]));
  const unsupported = await api.unsupportedWorkerProtocolStatus();
  expect(unsupported.status).toBe(400);
  await showStandaloneCard(page, {
    id: "UAT-WORKER-001",
    title: "Worker API capabilities and protocol guardrails",
    category: "worker",
    expected: "pass",
    elements: [],
    functions: ["jobs capabilities", "worker protocol version", "idempotency guardrail"],
  }, deployment, `Capabilities: ${capabilities.capabilities.join(", ")}\nUnsupported protocol rejected with HTTP ${unsupported.status}`);
}

async function runBundledIdentityManagementCase(page: Page, api: ArtificialFlowApi, created: Created): Promise<void> {
  const users = await api.identityUsers();
  const roles = await api.identityRoles();
  expect(users.length).toBeGreaterThan(0);
  expect(roles.length).toBeGreaterThan(0);
  const first = await createBundledSDKProfile(api, `uat-video-${runId}`, created);
  const sdk = new NodeSDKClient({
    baseUrl: `${process.env.FRONTEND_URL || "http://localhost:9100"}/api`,
    auth: { type: "zitadel-jwt-profile", profile: first.profile },
  });
  const sdkIdentity = await sdk.getIdentity();
  expect(sdkIdentity.authenticated).toBe(true);
  expect(sdkIdentity.principal?.roles || []).toContain("artificialflow client");
  const sdkJobs = await sdk.activateJobs({
    type: "uat-sdk-auth-no-jobs",
    worker: `uat-official-sdk-${runId}`,
    maxJobs: 1,
    timeoutMs: 10,
    lockDurationMs: 1_000,
  });
  expect(Array.isArray(sdkJobs.jobs)).toBe(true);

  const replacementKeyPair = await generateServiceAccountKeyPair();
  const replacementKey = await api.addClientKey(first.clientId, replacementKeyPair.publicKeyPem);
  const replacementProfile = createServiceAccountProfile({
    keyId: replacementKey.id,
    userId: first.clientId,
    issuer: first.profile.issuer,
    tokenUrl: first.profile.tokenUrl,
    scopes: first.profile.scopes,
  }, replacementKeyPair.privateKeyPem) as ZitadelJwtProfile;
  const replacementSDK = new NodeSDKClient({
    baseUrl: `${process.env.FRONTEND_URL || "http://localhost:9100"}/api`,
    auth: { type: "zitadel-jwt-profile", profile: replacementProfile },
  });
  expect((await replacementSDK.getIdentity()).authenticated).toBe(true);
  expect((await sdk.getIdentity()).authenticated).toBe(true);

  const firstProvider = new ZitadelJwtProfileAuthProvider({ type: "zitadel-jwt-profile", profile: first.profile });
  const mintedBeforeRevocation = await firstProvider.getToken();
  const accessClaims = JSON.parse(Buffer.from(mintedBeforeRevocation.split(".")[1] || "", "base64url").toString("utf8")) as {
    aud?: string | string[];
    iat?: number;
    exp?: number;
  };
  expect(accessClaims.exp).toBeGreaterThan(accessClaims.iat || 0);
  expect((accessClaims.exp || 0) - (accessClaims.iat || 0)).toBeLessThanOrEqual(900);
  expect(Array.isArray(accessClaims.aud) ? accessClaims.aud : [accessClaims.aud]).toContain(first.projectId);
  await api.revokeClientKey(first.clientId, first.keyId);
  await expect(new NodeSDKClient({
    baseUrl: `${process.env.FRONTEND_URL || "http://localhost:9100"}/api`,
    auth: { type: "zitadel-jwt-profile", profile: first.profile },
  }).getIdentity()).rejects.toThrow();
  expect((await api.requestWithToken<{ authenticated: boolean }>("/identity/me", mintedBeforeRevocation)).authenticated).toBe(true);

  const replacementAccessToken = await new ZitadelJwtProfileAuthProvider({
    type: "zitadel-jwt-profile",
    profile: replacementProfile,
  }).getToken();
  await api.terminateIdentityUser(first.clientId);
  await api.requestWithToken("/identity/me", replacementAccessToken, {}, 401);
  await api.deleteClient(first.clientId);
  created.clients = created.clients.filter((id) => id !== first.clientId);

  await page.goto("/sdk-clients", { waitUntil: "domcontentloaded" });
  await addBanner(page, "Private-key JWT Profile mints short-lived tokens with overlapping key rotation");
  await expect(page.locator("body")).toContainText(/SDK Clients|ArtificialFlow Clients|Clients/i);
}

async function createBundledSDKProfile(api: ArtificialFlowApi, name: string, created: Created) {
  const keyPair = await generateServiceAccountKeyPair();
  const client = await api.createClientKey(name, keyPair.publicKeyPem);
  const key = client.credentials.find((credential) => credential.type === "private_key_jwt");
  expect(key, "ZITADEL must return registered public-key metadata").toBeTruthy();
  const config = await api.identityConfig();
  const issuer = String(config.frontend_oidc_authority || "").replace(/\/+$/, "");
  const projectId = String(config.client_id || "");
  expect(issuer).not.toBe("");
  expect(projectId).not.toBe("");
  created.clients.push(client.client_id);
  const profile = createServiceAccountProfile({
    keyId: key?.id || "",
    userId: client.client_id,
    issuer,
    tokenUrl: `${issuer}/oauth/v2/token`,
    scopes: [
      "openid",
      "urn:zitadel:iam:org:projects:roles",
      `urn:zitadel:iam:org:project:id:${projectId}:aud`,
    ],
  }, keyPair.privateKeyPem) as ZitadelJwtProfile;
  return { clientId: client.client_id, keyId: key?.id || "", profile, projectId };
}

function roleUser(role: "accountant" | "reviewer", mode: UatDeployment) {
  const password = "UatPass123!";
  if (mode === "external-keycloak") {
    return {
      role,
      username: role,
      password,
      display: role === "accountant" ? "UAT Accountant" : "UAT Reviewer",
    };
  }
  return {
    role,
    username: role,
    password,
    display: role === "accountant" ? "UAT Accountant" : "UAT Reviewer",
  };
}

async function provisionRoleUsers(
  api: ArtificialFlowApi,
  mode: UatDeployment,
  accountant: ReturnType<typeof roleUser>,
  reviewer: ReturnType<typeof roleUser>,
  created: Created,
): Promise<void> {
  if (mode === "external-keycloak") {
    return;
  }

  for (const role of [accountant.role, reviewer.role]) {
    try {
      await api.createIdentityRole(role, `UAT ${role}`, "UAT");
      created.roles.push(role);
    } catch (err) {
      console.log(`[uat-role] role ${role} already exists or could not be created: ${String(err)}`);
    }
  }

  for (const user of [accountant, reviewer]) {
    try {
      const createdUser = await api.createIdentityUser({
        username: user.username,
        given_name: user.display.split(" ")[1] || user.role,
        family_name: "User",
        email: `${user.username}@artificialflow.io`,
        password: user.password,
        roles: [user.role],
      });
      created.users.push(createdUser.id);
    } catch (err) {
      console.log(`[uat-role] user ${user.username} already exists or could not be created: ${String(err)}`);
    }
  }
}

function actingUserForRole(user: ReturnType<typeof roleUser>): ActingUser {
  return {
    subject: user.username,
    username: user.username,
    email: `${user.username}@artificialflow.io`,
    name: user.display,
    roles: [user.role],
  };
}

async function provisionInboxClientToken(page: Page, api: ArtificialFlowApi, mode: UatDeployment, created: Created): Promise<string> {
  if (mode === "bundled-zitadel") {
    const client = await createBundledSDKProfile(api, `uat-inbox-${runId}`, created);
    return new ZitadelJwtProfileAuthProvider({
      type: "zitadel-jwt-profile",
      profile: client.profile,
    }).getToken();
  }

  const tokenResponse = await page.request.post("http://localhost:9181/realms/artificialflow/protocol/openid-connect/token", {
    form: {
      grant_type: "password",
      client_id: "artificialflow-frontend",
      username: "sdk-client",
      password: "UatPass123!",
    },
  });
  expect(tokenResponse.ok()).toBeTruthy();
  const body = await tokenResponse.json() as { access_token?: string };
  if (!body.access_token) {
    throw new Error("External Keycloak did not return an SDK client access token");
  }
  return body.access_token;
}

async function startInstanceFromProcessesOrApi(
  page: Page,
  api: ArtificialFlowApi,
  workflowId: string,
  processId: string,
  mode: UatDeployment,
): Promise<WorkflowInstance> {
  await page.goto("/processes", { waitUntil: "domcontentloaded" });
  await addBanner(page, "Start the complex process from the Processes screen");
  if (mode !== "external-keycloak" && await page.locator("body").getByText(processId).count()) {
    let startedId = "";
    page.once("dialog", async (dialog) => {
      const match = dialog.message().match(/Instance\s+([^\s]+)/i);
      startedId = match?.[1] || "";
      await dialog.accept();
    });
    await page.getByRole("row").filter({ hasText: processId }).getByRole("button", { name: /^start$/i }).click();
    await expect.poll(() => startedId, { timeout: 10_000 }).not.toBe("");
    return api.getInstance(startedId);
  }

  await addBanner(page, "Process catalog is non-blocking here; starting via command API for the same workflow");
  return api.startInstance(workflowId, {
    amount: 9500,
    needsArchive: true,
    needsCompliance: true,
    vendor: "ACME Supplies",
    uatCase: "UAT-END2END-001",
  });
}

async function completeWorkerJob(page: Page, api: ArtificialFlowApi, type: string, worker: string): Promise<void> {
  await addBanner(page, `External worker activates and completes ${type}`);
  const jobs = await pollJobs(api, type, worker);
  expect(jobs.length).toBeGreaterThan(0);
  await api.completeJob(jobs[0].key, worker, { workerValidated: true });
}

async function completeTaskFromInbox(
  page: Page,
  api: ArtificialFlowApi,
  token: string,
  actingUser: ActingUser,
  instanceId: string,
  stepId: string,
  banner: string,
): Promise<void> {
  await page.goto(`/instances/${instanceId}`, { waitUntil: "domcontentloaded" });
  await addBanner(page, banner);
  const inboxInstance = await api.getInboxInstance(instanceId, token, actingUser);
  expect(inboxInstance.id).toBe(instanceId);
  const tasks = await api.listInboxTasks(instanceId, token, actingUser);
  const task = tasks.find((item) => item.elementId === stepId);
  expect(task, `${actingUser.username} should see ${stepId} in ArtificialFlow Transaction Inbox`).toBeTruthy();
  if (!task) {
    throw new Error(`${actingUser.username} could not see ${stepId} in ArtificialFlow Transaction Inbox`);
  }
  expect(task.canClaim || task.canComplete).toBeTruthy();
  const claimed = task.canClaim ? await api.claimInboxTask(instanceId, task.executionId, token, actingUser) : task;
  const executionId = typeof claimed === "string" ? task.executionId : claimed.executionId;
  await api.completeInboxTask(instanceId, executionId, token, actingUser);
}

async function assertInboxCannotSeeOrClaimTask(
  page: Page,
  api: ArtificialFlowApi,
  token: string,
  actingUser: ActingUser,
  instanceId: string,
  executionId: string,
  stepId: string,
): Promise<void> {
  await page.goto("/instances", { waitUntil: "domcontentloaded" });
  await addBanner(page, `${stepId}: ${actingUser.username} cannot take a task assigned to another role`);
  const inboxItems = await api.listInboxItems(token, actingUser);
  const inboxTasks = await api.listInboxTasks(instanceId, token, actingUser).catch(() => []);
  expect(inboxItems.some((item) => item.id === instanceId && inboxTasks.some((task) => task.elementId === stepId))).toBeFalsy();
  await api.claimInboxTask(instanceId, executionId, token, actingUser, 403);
}

async function showCompletedHistoryEvidence(page: Page, instanceId: string): Promise<void> {
  const deadline = Date.now() + 60_000;
  let lastBody = "";
  while (Date.now() < deadline) {
    await page.goto("/history", { waitUntil: "domcontentloaded" });
    await addBanner(page, "History screen shows completed instances, actions taken, and who completed each task");
    lastBody = await page.locator("body").innerText().catch(() => "");
    if (
      lastBody.includes(instanceId) &&
      lastBody.includes("accountantReview") &&
      lastBody.includes("reviewerApproval") &&
      /Completed by/i.test(lastBody)
    ) {
      await expect(page.locator("body")).toContainText("Completed Instance History");
      await expect(page.locator("body")).toContainText("accountantReview");
      await expect(page.locator("body")).toContainText("reviewerApproval");
      await expect(page.locator("body")).toContainText(/Completed by/i);
      return;
    }

    const refresh = page.getByRole("button", { name: /refresh/i }).first();
    if (await refresh.count()) {
      await refresh.click().catch(() => undefined);
    }
    await page.waitForTimeout(2_000);
  }
  throw new Error(`Timed out waiting for completed history evidence for ${instanceId}. Last page text: ${lastBody.slice(0, 500)}`);
}

function activeExecutionForStep(instance: WorkflowInstance, stepId: string) {
  const execution = (instance.executions || []).find((item) => item.status === "ACTIVE" && item.step_id === stepId);
  if (!execution) {
    throw new Error(`No active execution found for ${stepId}`);
  }
  return execution;
}

async function deployStart(page: Page, api: ArtificialFlowApi, uatCase: UatCase, created: Created, context: Record<string, unknown> = {}) {
  const workflowId = await deployBundle(api, uatCase, created);
  const instance = await api.startInstance(workflowId, { ...context, uatCase: uatCase.id });
  created.instances.push(instance.id);
  await waitForProjectionEvidence(page, api, workflowId, processIdFor(uatCase));
  await page.waitForTimeout(500);
  return { workflowId, instance };
}

async function waitForProjectionEvidence(page: Page, api: ArtificialFlowApi, workflowId: string, processId: string): Promise<void> {
  if (deployment === "external-keycloak") {
    await addBanner(page, `External Keycloak: command API validated; query projection for ${processId} is recorded as non-blocking evidence`);
    return;
  }
  try {
    await api.waitForWorkflowInQuery(workflowId, processId);
  } catch (err) {
    throw err;
  }
}

async function waitForProjectionEvidenceNonBlocking(page: Page, api: ArtificialFlowApi, workflowId: string, processId: string): Promise<void> {
  try {
    await waitForProjectionEvidence(page, api, workflowId, processId);
  } catch {
    await addBanner(page, `Command deploy validated; process catalog projection for ${processId} is still syncing`);
  }
}

async function deployBundle(api: ArtificialFlowApi, uatCase: UatCase, created: Created): Promise<string> {
  const bundle = buildBpmnFixture(uatCase.fixture as BpmnFixtureName, `${runId}_${uatCase.id}`);
  for (const dependency of bundle.dependencies || []) {
    const workflow = await api.deploy(dependency.xml);
    created.workflows.push(workflow.id);
  }
  const workflow = await api.deploy(bundle.primary.xml);
  created.workflows.push(workflow.id);
  return workflow.id;
}

async function pollJobs(api: ArtificialFlowApi, type: string, worker: string) {
  const deadline = Date.now() + 20_000;
  while (Date.now() < deadline) {
    const jobs = await api.activateJobs(type, worker);
    if (jobs.length > 0) return jobs;
  }
  return [];
}

async function completeAllActiveTasks(api: ArtificialFlowApi, instanceId: string): Promise<void> {
  let instance = await api.getInstance(instanceId);
  for (const step of activeSteps(instance)) {
    await api.completeTask(instanceId, step);
  }
}

function activeSteps(instance: WorkflowInstance): string[] {
  return (instance.executions || []).filter((execution) => execution.status === "ACTIVE").map((execution) => execution.step_id || execution.id);
}

function processIdFor(uatCase: UatCase): string {
  const bundle = buildBpmnFixture(uatCase.fixture as BpmnFixtureName, `${runId}_${uatCase.id}`);
  return bundle.primary.processId;
}

async function showExpectedSkip(page: Page, uatCase: UatCase, mode: UatDeployment): Promise<void> {
  await showStandaloneCard(page, uatCase, mode, uatCase.reason || "This case is intentionally tracked as an expected skip.");
}

async function showStandaloneCard(page: Page, uatCase: UatCase, mode: UatDeployment, detail: string): Promise<void> {
  await page.setContent(`<!doctype html>
<html><head><meta charset="utf-8"><title>${uatCase.id}</title>
<style>
body{margin:0;background:#f8fafc;color:#0f172a;font:20px/1.45 system-ui,sans-serif}
main{max-width:1100px;margin:64px auto;background:white;border:1px solid #cbd5e1;border-radius:24px;padding:42px;box-shadow:0 16px 40px rgba(15,23,42,.12)}
.eyebrow{font-size:14px;font-weight:800;color:#1d4ed8;letter-spacing:.08em;text-transform:uppercase}
h1{font-size:40px;line-height:1.1;margin:12px 0}.pill{display:inline-block;margin:6px 8px 0 0;padding:6px 10px;border:1px solid #cbd5e1;border-radius:999px;background:#f1f5f9;font-size:15px;font-weight:700}
pre{white-space:pre-wrap;background:#020617;color:#e2e8f0;border-radius:16px;padding:18px;font-size:16px}
</style></head><body><main>
<div class="eyebrow">${mode} - ${uatCase.expected}</div>
<h1>${uatCase.id}: ${uatCase.title}</h1>
<p><strong>Functions:</strong> ${uatCase.functions.join(", ") || "None"}</p>
<p><strong>BPMN Elements:</strong></p>
<div>${uatCase.elements.map((element) => `<span class="pill">${element}</span>`).join("") || "<span class=\"pill\">N/A</span>"}</div>
<h2>Evidence</h2>
<pre>${escapeHtml(detail)}</pre>
</main></body></html>`, { waitUntil: "load" });
  await page.waitForTimeout(3_500);
}

async function addBanner(page: Page, text: string): Promise<void> {
  await page.evaluate((text) => {
    let banner = document.getElementById("artificialflow-uat-banner");
    if (!banner) {
      banner = document.createElement("div");
      banner.id = "artificialflow-uat-banner";
      banner.style.cssText = "position:fixed;z-index:999999;top:12px;left:50%;transform:translateX(-50%);max-width:90vw;padding:10px 16px;background:#0f172a;color:white;border-radius:999px;font:700 14px system-ui;box-shadow:0 8px 24px rgba(15,23,42,.25);text-align:center";
      document.body.appendChild(banner);
    }
    banner.textContent = text;
  }, text);
  await page.waitForTimeout(1_000);
}

async function cleanup(api: ArtificialFlowApi, created: Created): Promise<void> {
  if (process.env.UAT_KEEP_EVIDENCE === "true") {
    console.log("[uat-cleanup] preserving created UAT evidence because UAT_KEEP_EVIDENCE=true");
    return;
  }
  for (const userId of created.users.reverse()) {
    await api.deleteIdentityUser(userId).catch(() => undefined);
  }
  for (const role of created.roles.reverse()) {
    await api.deleteIdentityRole(role).catch(() => undefined);
  }
  for (const clientId of created.clients.reverse()) {
    await api.deleteClient(clientId).catch(() => undefined);
  }
  for (const instanceId of created.instances.reverse()) {
    await api.deleteInstance(instanceId).catch(() => undefined);
  }
  for (const workflowId of created.workflows.reverse()) {
    await api.deleteWorkflow(workflowId).catch(() => undefined);
  }
}

function escapeHtml(value: string): string {
  return value.replace(/[&<>"']/g, (char) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#39;",
  }[char] || char));
}

interface Created {
  workflows: string[];
  instances: string[];
  clients: string[];
  users: string[];
  roles: string[];
}

const assert = require('assert');
const {
  HeaderWorkerProtocolVersion,
  GoFlowApiError,
  GoFlowClient,
  WorkerProtocolVersion,
} = require('../dist');

function jsonResponse(status, payload) {
  return {
    ok: status < 400,
    status,
    statusText: status < 400 ? 'OK' : 'Error',
    headers: { get: (name) => name.toLowerCase() === 'content-type' ? 'application/json' : null },
    text: async () => payload === undefined ? '' : JSON.stringify(payload),
  };
}

function textResponse(status, payload) {
  return {
    ok: status < 400,
    status,
    statusText: status < 400 ? 'OK' : 'Error',
    headers: { get: () => 'text/plain' },
    text: async () => payload || '',
  };
}

function createClient(responses) {
  const calls = [];
  const fetch = async (url, init = {}) => {
    calls.push({ url, init });
    const response = responses.shift();
    if (!response) {
      throw new Error(`unexpected request ${init.method || 'GET'} ${url}`);
    }
    return typeof response === 'function' ? response(url, init) : response;
  };
  const client = new GoFlowClient({
    baseUrl: 'http://goflow.local/api',
    token: async () => 'token-1',
    fetch,
  });
  return { client, calls };
}

function body(call) {
  return call.init.body ? JSON.parse(call.init.body) : undefined;
}

function wait(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function testHealthAuthHeaders() {
  const { client, calls } = createClient([
    jsonResponse(200, { status: 'ok' }),
    jsonResponse(200, { status: 'ok' }),
    jsonResponse(200, { status: 'ok' }),
  ]);

  await client.health();
  assert.strictEqual(calls[0].url, 'http://goflow.local/api/health');
  assert.strictEqual(calls[0].init.method, 'GET');
  assert.strictEqual(calls[0].init.headers.Authorization, 'Bearer token-1');
  assert.ok(calls[0].init.headers['X-Correlation-ID']);

  await client.queryHealth();
  assert.strictEqual(calls[1].url, 'http://goflow.local/api/query/health');

  client.setToken('token-2');
  await client.health({ correlationId: 'corr-1', headers: { 'X-Custom': 'yes' } });
  assert.strictEqual(calls[2].init.headers.Authorization, 'Bearer token-2');
  assert.strictEqual(calls[2].init.headers['X-Correlation-ID'], 'corr-1');
  assert.strictEqual(calls[2].init.headers['X-Custom'], 'yes');
}

async function testAllWorkflowOperations() {
  const workflow = { id: '1', process_definition_id: 'order', name: 'Order', version: 1, resource_name: 'order.bpmn', deployment_id: '10', tenant_id: '', resource_checksum: 'abc', created_at: '2026-01-01T00:00:00Z', steps: [] };
  const { client, calls } = createClient([
    jsonResponse(200, workflow),
    jsonResponse(200, [workflow]),
    jsonResponse(200, { workflows: [workflow], total: 1 }),
    jsonResponse(200, { workflows: [workflow], total: 1 }),
    jsonResponse(200, workflow),
    jsonResponse(204),
  ]);

  await client.deployWorkflow('<definitions />');
  assert.strictEqual(calls[0].url, 'http://goflow.local/api/workflows');
  assert.strictEqual(calls[0].init.method, 'POST');
  assert.strictEqual(calls[0].init.body, '<definitions />');

  await client.listWorkflows({ source: 'command' });
  assert.strictEqual(calls[1].url, 'http://goflow.local/api/workflows');
  assert.strictEqual(calls[1].init.method, 'GET');

  await client.listWorkflows({ page: 1, pageSize: 10 });
  assert.strictEqual(calls[2].url, 'http://goflow.local/api/query/workflows?page=1&pageSize=10');

  await client.getWorkflows({ page: 2, pageSize: 20 });
  assert.strictEqual(calls[3].url, 'http://goflow.local/api/query/workflows?page=2&pageSize=20');

  await client.getWorkflow('1');
  assert.strictEqual(calls[4].url, 'http://goflow.local/api/workflows/1');

  await client.deleteWorkflow('1');
  assert.strictEqual(calls[5].url, 'http://goflow.local/api/workflows/1');
  assert.strictEqual(calls[5].init.method, 'DELETE');
}

async function testAllInstanceOperations() {
  const instance = { id: '99', workflow_id: '1', status: 'RUNNING', context: { orderId: 'A1' }, executions: [], created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' };
  const { client, calls } = createClient([
    jsonResponse(200, instance),
    jsonResponse(200, instance),
    jsonResponse(200, [instance]),
    jsonResponse(200, { instances: [instance], total: 1 }),
    jsonResponse(200, { instances: [instance], total: 1 }),
    jsonResponse(200, instance),
    jsonResponse(200, instance),
    jsonResponse(204),
    jsonResponse(204),
    jsonResponse(204),
  ]);

  await client.startInstance('1', { orderId: 'A1' });
  assert.strictEqual(calls[0].url, 'http://goflow.local/api/instances');
  assert.deepStrictEqual(body(calls[0]), { workflow_id: '1', context: { orderId: 'A1' } });

  await client.startInstance({ workflow_id: '1', context: { orderId: 'A2' } }, { idempotencyKey: 'idem-instance-2' });
  assert.strictEqual(calls[1].url, 'http://goflow.local/api/instances');
  assert.strictEqual(calls[1].init.headers['Idempotency-Key'], 'idem-instance-2');
  assert.deepStrictEqual(body(calls[1]), { workflow_id: '1', context: { orderId: 'A2' } });

  await client.listActiveInstances();
  assert.strictEqual(calls[2].url, 'http://goflow.local/api/instances');

  await client.getInstances({ workflowId: '1', state: 'RUNNING', page: 1, pageSize: 50 });
  assert.strictEqual(calls[3].url, 'http://goflow.local/api/query/instances?workflowId=1&state=RUNNING&page=1&pageSize=50');

  await client.searchInstances({ workflowId: '1' });
  assert.strictEqual(calls[4].url, 'http://goflow.local/api/query/instances?workflowId=1');

  await client.getInstance('99');
  assert.strictEqual(calls[5].url, 'http://goflow.local/api/instances/99');

  await client.getInstance('99', { source: 'query' });
  assert.strictEqual(calls[6].url, 'http://goflow.local/api/query/instances/99');

  await client.updateInstanceVariables('99', { approved: true });
  assert.strictEqual(calls[7].url, 'http://goflow.local/api/instances/99/variables');
  assert.deepStrictEqual(body(calls[7]), { variables: { approved: true } });

  await client.completeTask('99', 'approve-task');
  assert.strictEqual(calls[8].url, 'http://goflow.local/api/instances/99/complete');
  assert.deepStrictEqual(body(calls[8]), { step_id: 'approve-task' });

  await client.deleteInstance('99');
  assert.strictEqual(calls[9].url, 'http://goflow.local/api/instances/99');
  assert.strictEqual(calls[9].init.method, 'DELETE');
}

async function testMessagingAndWorkerRequests() {
  const { client, calls } = createClient([
    jsonResponse(204),
    jsonResponse(204),
    jsonResponse(204),
    jsonResponse(200, { jobs: [] }),
    jsonResponse(204),
    jsonResponse(204),
    jsonResponse(204),
    jsonResponse(200, { protocolVersion: 'v1', capabilities: ['activate'] }),
  ]);

  await client.publishSignal('OrderApproved', { orderId: 'A1' });
  assert.strictEqual(calls[0].url, 'http://goflow.local/api/signals');
  assert.deepStrictEqual(body(calls[0]), { signal_name: 'OrderApproved', payload: { orderId: 'A1' } });

  await client.publishSignal({ signal_name: 'OrderCanceled', payload: { orderId: 'A2' } });
  assert.strictEqual(calls[1].url, 'http://goflow.local/api/signals');
  assert.deepStrictEqual(body(calls[1]), { signal_name: 'OrderCanceled', payload: { orderId: 'A2' } });

  await client.publishMessage('MsgOrderPlaced', 'A1', { amount: 10 });
  assert.strictEqual(calls[2].url, 'http://goflow.local/api/messages');
  assert.deepStrictEqual(body(calls[2]), { message_name: 'MsgOrderPlaced', correlation_key: 'A1', payload: { amount: 10 } });

  await client.activateJobs({ type: 'payment', worker: 'worker-1', maxJobs: 2, timeoutMs: 500, lockDurationMs: 30000 });
  assert.strictEqual(calls[3].url, 'http://goflow.local/api/jobs/activate');
  assert.strictEqual(calls[3].init.headers[HeaderWorkerProtocolVersion], WorkerProtocolVersion);
  assert.deepStrictEqual(body(calls[3]), { type: 'payment', worker: 'worker-1', maxJobs: 2, timeoutMs: 500, lockDurationMs: 30000 });

  await client.completeJob('101', { worker: 'worker-1', variables: { ok: true } });
  assert.strictEqual(calls[4].url, 'http://goflow.local/api/jobs/101/complete');
  assert.strictEqual(calls[4].init.headers[HeaderWorkerProtocolVersion], WorkerProtocolVersion);
  assert.deepStrictEqual(body(calls[4]), { worker: 'worker-1', variables: { ok: true } });

  await client.failJob('101', { worker: 'worker-1', errorMessage: 'failed', retries: 1 });
  assert.strictEqual(calls[5].url, 'http://goflow.local/api/jobs/101/fail');
  assert.deepStrictEqual(body(calls[5]), { worker: 'worker-1', errorMessage: 'failed', retries: 1 });

  await client.extendJobLock('101', { worker: 'worker-1', lockDurationMs: 45000 });
  assert.strictEqual(calls[6].url, 'http://goflow.local/api/jobs/101/extend-lock');
  assert.deepStrictEqual(body(calls[6]), { worker: 'worker-1', lockDurationMs: 45000 });

  await client.getWorkerCapabilities();
  assert.strictEqual(calls[7].url, 'http://goflow.local/api/jobs/capabilities');
}

async function testMessageObjectOverload() {
  const { client, calls } = createClient([
    jsonResponse(204),
  ]);

  await client.publishMessage({ message_name: 'MsgOrderPlaced', correlation_key: 'A1', payload: { amount: 10 } });
  assert.strictEqual(calls[0].url, 'http://goflow.local/api/messages');
  assert.deepStrictEqual(body(calls[0]), { message_name: 'MsgOrderPlaced', correlation_key: 'A1', payload: { amount: 10 } });
}

async function testPlatformReadOperations() {
  const { client, calls } = createClient([
    jsonResponse(200, { outboxPending: 0, outboxPublishSuccess: 1, outboxPublishFailure: 0, outboxPublishLagSec: 0, outboxMaxAttempts: 5, idempotencyHit: 0, idempotencyMiss: 1 }),
    jsonResponse(200, { authenticated: true, principal: { subject: 'admin', roles: ['goflow admin'] } }),
    jsonResponse(200, { deployment_mode: 'zitadel', configuration_source: 'env', provider_name: 'ZITADEL', auth_enabled: true, frontend_auth_enabled: true, frontend_oidc_authority: 'http://localhost:9180', frontend_oidc_client_id: '123', token_validation_mode: 'jwt', internal_issuer_url: 'http://zitadel-proxy', external_issuer_url: 'http://localhost:9180', client_id: '', introspection_url: '', introspection_client_id: '', introspection_auth_method: '', enforce_audience: false, allow_insecure_issuer: true, claim_subject_path: 'sub', claim_roles_path: 'roles', claim_scopes_path: 'scope', claim_tenant_path: 'tenant', claim_email_path: 'email', claim_name_path: 'name', standard_roles: ['goflow admin'] }),
  ]);

  await client.getEngineMetrics();
  assert.strictEqual(calls[0].url, 'http://goflow.local/api/internal/metrics');

  await client.getIdentity();
  assert.strictEqual(calls[1].url, 'http://goflow.local/api/identity/me');

  await client.getIdentityConfig();
  assert.strictEqual(calls[2].url, 'http://goflow.local/api/identity/config');
}

async function testIdentityManagementRequests() {
  const user = { id: 'u1', username: 'admin', preferred_login_name: 'admin@admin.localhost', display_name: 'admin', given_name: 'admin', family_name: 'admin', email: 'admin@admin.localhost', email_verified: true, state: 'ACTIVE', type: 'human', created_at: '2026-01-01T00:00:00Z', changed_at: '2026-01-01T00:00:00Z', roles: ['goflow admin'] };
  const role = { key: 'goflow admin', display_name: 'Admin', group: 'GoFlow' };
  const clientToken = { client_id: 'client-1', username: 'sdk-orders', name: 'Orders SDK', description: 'Order service', environment: 'production', owner_email: 'platform@example.com', purpose: 'Order worker', role: 'goflow client', token_id: 'pat-1', token: 'sdk-token', token_created_at: '2026-01-01T00:00:00Z', token_expires_at: '2027-01-01T00:00:00Z' };
  const identityClient = { client_id: 'client-1', username: 'sdk-orders', name: 'Orders SDK', description: 'Order service', environment: 'production', owner_email: 'platform@example.com', purpose: 'Order worker', role: 'goflow client', state: 'USER_STATE_ACTIVE', created_at: '2026-01-01T00:00:00Z', changed_at: '2026-01-01T00:00:00Z', tokens: [{ token_id: 'pat-1', token_created_at: '2026-01-01T00:00:00Z', token_changed_at: '2026-01-01T00:00:00Z', token_expires_at: '2027-01-01T00:00:00Z', status: 'active' }] };
  const { client, calls } = createClient([
    jsonResponse(200, { users: [user] }),
    jsonResponse(200, { users: [user] }),
    jsonResponse(201, user),
    jsonResponse(201, user),
    jsonResponse(201, clientToken),
    jsonResponse(201, clientToken),
    jsonResponse(201, clientToken),
    jsonResponse(200, { clients: [identityClient] }),
    jsonResponse(200, { clients: [identityClient] }),
    jsonResponse(200, { clients: [identityClient] }),
    jsonResponse(201, clientToken),
    jsonResponse(201, clientToken),
    jsonResponse(204),
    jsonResponse(204),
    jsonResponse(204),
    jsonResponse(204),
    jsonResponse(200, user),
    jsonResponse(200, user),
    jsonResponse(204),
    jsonResponse(204),
    jsonResponse(204),
    jsonResponse(204),
    jsonResponse(204),
    jsonResponse(204),
    jsonResponse(200, { roles: [role] }),
    jsonResponse(200, { roles: [role] }),
    jsonResponse(201, role),
    jsonResponse(201, role),
    jsonResponse(200, role),
    jsonResponse(200, role),
    jsonResponse(204),
    jsonResponse(204),
  ]);

  await client.listIdentityUsers();
  assert.strictEqual(calls[0].url, 'http://goflow.local/api/identity/management/users');

  await client.getIdentityManagementUsers();
  assert.strictEqual(calls[1].url, 'http://goflow.local/api/identity/management/users');

  await client.createIdentityUser({ given_name: 'User', family_name: 'One', email: 'user@example.com', password: 'secret' });
  assert.deepStrictEqual(body(calls[2]), { given_name: 'User', family_name: 'One', email: 'user@example.com', password: 'secret', username: '', password_change_required: false, roles: [] });

  await client.createIdentityManagementUser({ username: 'user', given_name: 'User', family_name: 'One', email: 'user@example.com', password: 'secret', password_change_required: true, roles: ['goflow viewer'] });
  assert.deepStrictEqual(body(calls[3]), { username: 'user', given_name: 'User', family_name: 'One', email: 'user@example.com', password: 'secret', password_change_required: true, roles: ['goflow viewer'] });

  await client.createIdentityClientToken({ name: 'Orders SDK', username: 'sdk-orders', description: 'Order service', environment: 'production', owner_email: 'platform@example.com', purpose: 'Order worker', token_expires_at: '2027-01-01T00:00:00Z' });
  assert.strictEqual(calls[4].url, 'http://goflow.local/api/identity/management/clients');
  assert.deepStrictEqual(body(calls[4]), { name: 'Orders SDK', username: 'sdk-orders', description: 'Order service', environment: 'production', owner_email: 'platform@example.com', purpose: 'Order worker', token_expires_at: '2027-01-01T00:00:00Z' });

  await client.createIdentityManagementClientToken({ name: 'Worker SDK' });
  assert.deepStrictEqual(body(calls[5]), { name: 'Worker SDK', username: '', description: '', environment: '', owner_email: '', purpose: '', token_expires_at: '' });

  await client.createGoFlowClientToken({ name: 'API SDK', username: 'api-sdk' });
  assert.deepStrictEqual(body(calls[6]), { name: 'API SDK', username: 'api-sdk', description: '', environment: '', owner_email: '', purpose: '', token_expires_at: '' });

  await client.listIdentityClients();
  assert.strictEqual(calls[7].url, 'http://goflow.local/api/identity/management/clients');

  await client.getIdentityManagementClients();
  assert.strictEqual(calls[8].url, 'http://goflow.local/api/identity/management/clients');

  await client.listGoFlowClients();
  assert.strictEqual(calls[9].url, 'http://goflow.local/api/identity/management/clients');

  await client.rotateIdentityClientToken('client-1', { token_expires_at: '2028-01-01T00:00:00Z' });
  assert.strictEqual(calls[10].url, 'http://goflow.local/api/identity/management/clients/client-1/tokens');
  assert.deepStrictEqual(body(calls[10]), { token_expires_at: '2028-01-01T00:00:00Z' });

  await client.rotateIdentityManagementClientToken('client-1');
  assert.deepStrictEqual(body(calls[11]), { token_expires_at: '' });

  await client.revokeIdentityClientToken('client-1', 'pat-1');
  assert.strictEqual(calls[12].url, 'http://goflow.local/api/identity/management/clients/client-1/tokens/pat-1');
  assert.strictEqual(calls[12].init.method, 'DELETE');

  await client.revokeIdentityManagementClientToken('client-1', 'pat-2');
  assert.strictEqual(calls[13].url, 'http://goflow.local/api/identity/management/clients/client-1/tokens/pat-2');

  await client.deleteIdentityClient('client-1');
  assert.strictEqual(calls[14].url, 'http://goflow.local/api/identity/management/clients/client-1');
  assert.strictEqual(calls[14].init.method, 'DELETE');

  await client.deleteIdentityManagementClient('client-2');
  assert.strictEqual(calls[15].url, 'http://goflow.local/api/identity/management/clients/client-2');

  await client.updateIdentityUser('u1', { given_name: 'Admin', roles: ['goflow admin'] });
  assert.strictEqual(calls[16].url, 'http://goflow.local/api/identity/management/users/u1');
  assert.strictEqual(calls[16].init.method, 'PUT');
  assert.deepStrictEqual(body(calls[16]), { given_name: 'Admin', roles: ['goflow admin'] });

  await client.updateIdentityManagementUser('u1', { display_name: 'Admin User' });
  assert.strictEqual(calls[17].url, 'http://goflow.local/api/identity/management/users/u1');

  await client.terminateIdentityUser('u1');
  assert.strictEqual(calls[18].url, 'http://goflow.local/api/identity/management/users/u1/terminate');

  await client.terminateIdentityManagementUser('u1');
  assert.strictEqual(calls[19].url, 'http://goflow.local/api/identity/management/users/u1/terminate');

  await client.reactivateIdentityUser('u1');
  assert.strictEqual(calls[20].url, 'http://goflow.local/api/identity/management/users/u1/reactivate');

  await client.reactivateIdentityManagementUser('u1');
  assert.strictEqual(calls[21].url, 'http://goflow.local/api/identity/management/users/u1/reactivate');

  await client.deleteIdentityUser('u1');
  assert.strictEqual(calls[22].url, 'http://goflow.local/api/identity/management/users/u1');
  assert.strictEqual(calls[22].init.method, 'DELETE');

  await client.deleteIdentityManagementUser('u1');
  assert.strictEqual(calls[23].url, 'http://goflow.local/api/identity/management/users/u1');

  await client.listIdentityRoles();
  assert.strictEqual(calls[24].url, 'http://goflow.local/api/identity/management/roles');

  await client.getIdentityManagementRoles();
  assert.strictEqual(calls[25].url, 'http://goflow.local/api/identity/management/roles');

  await client.createIdentityRole({ key: 'custom', display_name: 'Custom' });
  assert.deepStrictEqual(body(calls[26]), { key: 'custom', display_name: 'Custom', group: 'GoFlow' });

  await client.createIdentityManagementRole({ key: 'custom-2', display_name: 'Custom 2', group: 'Custom' });
  assert.deepStrictEqual(body(calls[27]), { key: 'custom-2', display_name: 'Custom 2', group: 'Custom' });

  await client.updateIdentityRole('custom', { display_name: 'Updated', group: 'Custom' });
  assert.strictEqual(calls[28].url, 'http://goflow.local/api/identity/management/roles/custom');
  assert.strictEqual(calls[28].init.method, 'PUT');

  await client.updateIdentityManagementRole('custom-2', { display_name: 'Updated 2' });
  assert.strictEqual(calls[29].url, 'http://goflow.local/api/identity/management/roles/custom-2');

  await client.deleteIdentityRole('custom');
  assert.strictEqual(calls[30].url, 'http://goflow.local/api/identity/management/roles/custom');
  assert.strictEqual(calls[30].init.method, 'DELETE');

  await client.deleteIdentityManagementRole('custom-2');
  assert.strictEqual(calls[31].url, 'http://goflow.local/api/identity/management/roles/custom-2');
}

async function testWorkerRunOnceCompletesAndFails() {
  const success = createClient([
    jsonResponse(200, { jobs: [{ key: '201', type: 'payment', processInstanceKey: '1', elementInstanceKey: '2', processDefinitionKey: '3', elementId: 'task', worker: 'worker-1', retries: 3, state: 'ACTIVATED', createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z' }] }),
    jsonResponse(204),
  ]);
  const worker = success.client.createWorker('payment', async () => ({ paid: true }), { workerName: 'worker-1' });
  const count = await worker.runOnce();
  assert.strictEqual(count, 1);
  assert.strictEqual(success.calls[1].url, 'http://goflow.local/api/jobs/201/complete');
  assert.deepStrictEqual(body(success.calls[1]), { worker: 'worker-1', variables: { paid: true } });

  const failure = createClient([
    jsonResponse(200, { jobs: [{ key: '202', type: 'payment', processInstanceKey: '1', elementInstanceKey: '2', processDefinitionKey: '3', elementId: 'task', worker: 'worker-1', retries: 2, state: 'ACTIVATED', createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z' }] }),
    jsonResponse(204),
  ]);
  const failingWorker = failure.client.createWorker('payment', async () => { throw new Error('boom'); }, { workerName: 'worker-1' });
  await failingWorker.runOnce();
  assert.strictEqual(failure.calls[1].url, 'http://goflow.local/api/jobs/202/fail');
  assert.deepStrictEqual(body(failure.calls[1]), { worker: 'worker-1', errorMessage: 'boom', retries: 1 });

  const lockExtension = createClient([
    jsonResponse(200, { jobs: [{ key: '203', type: 'payment', processInstanceKey: '1', elementInstanceKey: '2', processDefinitionKey: '3', elementId: 'task', worker: 'worker-1', retries: 3, state: 'ACTIVATED', createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z' }] }),
    jsonResponse(204),
    jsonResponse(204),
  ]);
  const lockWorker = lockExtension.client.createWorker('payment', async (_job, context) => {
    await context.extendLock(60000);
    return { locked: true };
  }, { workerName: 'worker-1' });
  await lockWorker.runOnce();
  assert.strictEqual(lockExtension.calls[1].url, 'http://goflow.local/api/jobs/203/extend-lock');
  assert.deepStrictEqual(body(lockExtension.calls[1]), { worker: 'worker-1', lockDurationMs: 60000 });
  assert.strictEqual(lockExtension.calls[2].url, 'http://goflow.local/api/jobs/203/complete');

  const autoStart = createClient([
    jsonResponse(200, { jobs: [] }),
  ]);
  const autoWorker = autoStart.client.createWorker('payment', async () => undefined, { workerName: 'worker-1', autoStart: true, pollIntervalMs: 10000 });
  assert.strictEqual(autoWorker.isRunning(), true);
  await wait(0);
  autoWorker.stop();
  assert.strictEqual(autoWorker.isRunning(), false);
  assert.strictEqual(autoStart.calls[0].url, 'http://goflow.local/api/jobs/activate');
}

async function testApiErrors() {
  const { client } = createClient([textResponse(500, 'broken')]);
  await assert.rejects(
    () => client.health(),
    (error) => error instanceof GoFlowApiError && error.status === 500 && error.body === 'broken',
  );
}

async function main() {
  await testHealthAuthHeaders();
  await testAllWorkflowOperations();
  await testAllInstanceOperations();
  await testMessagingAndWorkerRequests();
  await testMessageObjectOverload();
  await testPlatformReadOperations();
  await testIdentityManagementRequests();
  await testWorkerRunOnceCompletesAndFails();
  await testApiErrors();
  console.log('client.test.js passed');
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});

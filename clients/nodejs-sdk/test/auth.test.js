const assert = require('assert');
const { generateKeyPairSync, verify } = require('node:crypto');
const {
  ArtificialFlowClient,
  OAuthClientCredentialsAuthError,
  OAuthClientCredentialsAuthProvider,
  ZitadelJwtProfileAuthError,
  ZitadelJwtProfileAuthProvider,
} = require('../dist');

function createProfile(modulusLength = 2048) {
  const { privateKey, publicKey } = generateKeyPairSync('rsa', {
    modulusLength,
    publicKeyEncoding: { type: 'spki', format: 'pem' },
    privateKeyEncoding: { type: 'pkcs8', format: 'pem' },
  });
  return {
    profile: {
      type: 'serviceaccount',
      keyId: 'key-123',
      key: privateKey,
      userId: 'user-456',
      issuer: 'https://issuer.example',
      tokenUrl: 'https://issuer.example/oauth/v2/token',
      scopes: [
        'openid',
        'urn:zitadel:iam:org:projects:roles',
        'urn:zitadel:iam:org:project:id:project-789:aud',
      ],
    },
    publicKey,
  };
}

function response(status, payload, statusText = status < 400 ? 'OK' : 'Error') {
  return {
    ok: status < 400,
    status,
    statusText,
    headers: { get: () => 'application/json' },
    text: async () => payload === undefined ? '' : JSON.stringify(payload),
  };
}

function decodePart(value) {
  return JSON.parse(Buffer.from(value, 'base64url').toString('utf8'));
}

async function testExactSignedExchangeAndCache() {
  const { profile, publicKey } = createProfile();
  let now = 1_700_000_000_000;
  const calls = [];
  const tokens = ['access-one', 'access-two'];
  const provider = new ZitadelJwtProfileAuthProvider({
    type: 'zitadel-jwt-profile',
    profile,
    clock: () => now,
    refreshSkewMs: 10_000,
    fetch: async (url, init) => {
      calls.push({ url, init });
      return response(200, {
        access_token: tokens.shift(),
        token_type: 'Bearer',
        expires_in: 60,
      });
    },
  });

  assert.strictEqual(await provider.getToken(), 'access-one');
  assert.strictEqual(await provider.getToken(), 'access-one');
  assert.strictEqual(calls.length, 1);
  assert.strictEqual(calls[0].url, profile.tokenUrl);
  assert.strictEqual(calls[0].init.method, 'POST');
  assert.deepStrictEqual(calls[0].init.headers, {
    Accept: 'application/json',
    'Content-Type': 'application/x-www-form-urlencoded',
  });

  const form = new URLSearchParams(calls[0].init.body);
  assert.deepStrictEqual([...form.keys()].sort(), ['assertion', 'grant_type', 'scope']);
  assert.strictEqual(form.get('grant_type'), 'urn:ietf:params:oauth:grant-type:jwt-bearer');
  assert.strictEqual(form.get('scope'), profile.scopes.join(' '));

  const assertion = form.get('assertion');
  const parts = assertion.split('.');
  assert.strictEqual(parts.length, 3);
  assert.deepStrictEqual(decodePart(parts[0]), {
    alg: 'RS256',
    kid: profile.keyId,
    typ: 'JWT',
  });
  assert.deepStrictEqual(decodePart(parts[1]), {
    iss: profile.userId,
    sub: profile.userId,
    aud: profile.issuer,
    iat: 1_700_000_000,
    exp: 1_700_000_300,
  });
  assert.strictEqual(
    verify(
      'RSA-SHA256',
      Buffer.from(`${parts[0]}.${parts[1]}`),
      publicKey,
      Buffer.from(parts[2], 'base64url'),
    ),
    true,
  );

  now += 49_999;
  assert.strictEqual(await provider.getToken(), 'access-one');
  now += 2;
  assert.strictEqual(await provider.getToken(), 'access-two');
  assert.strictEqual(calls.length, 2);
}

async function testSingleFlightAndFailureRecovery() {
  const { profile } = createProfile();
  let release;
  let calls = 0;
  const provider = new ZitadelJwtProfileAuthProvider({
    type: 'zitadel-jwt-profile',
    profile,
    fetch: async () => {
      calls += 1;
      await new Promise((resolve) => { release = resolve; });
      return response(200, {
        access_token: 'shared-access-token',
        token_type: 'bearer',
        expires_in: 60,
      });
    },
  });

  const pending = [
    provider.getToken(),
    provider.getToken(),
    provider.getToken(),
  ];
  await Promise.resolve();
  assert.strictEqual(calls, 1);
  release();
  assert.deepStrictEqual(await Promise.all(pending), [
    'shared-access-token',
    'shared-access-token',
    'shared-access-token',
  ]);

  let attempt = 0;
  let rejectedBodyReads = 0;
  const retrying = new ZitadelJwtProfileAuthProvider({
    type: 'zitadel-jwt-profile',
    profile,
    fetch: async () => {
      attempt += 1;
      return attempt === 1
        ? {
          ...response(401, {}),
          text: async () => {
            rejectedBodyReads += 1;
            return JSON.stringify({
              error: 'invalid_grant',
              private_key: profile.key,
              assertion: 'secret-assertion',
            });
          },
        }
        : response(200, {
          access_token: 'recovered-token',
          expires_in: 60,
        });
    },
  });
  await assert.rejects(
    () => retrying.getToken(),
    (error) => {
      assert.ok(error instanceof ZitadelJwtProfileAuthError);
      assert.strictEqual(error.status, 401);
      assert.strictEqual(error.message, 'ZITADEL token exchange was rejected');
      assert.ok(!error.message.includes(profile.key));
      assert.ok(!error.message.includes('secret-assertion'));
      return true;
    },
  );
  assert.strictEqual(rejectedBodyReads, 0);
  assert.strictEqual(await retrying.getToken(), 'recovered-token');
  assert.strictEqual(attempt, 2);
}

async function testProfileAndResponseValidation() {
  const { profile } = createProfile();
  const { profile: weakProfile } = createProfile(1024);
  const invalidProfiles = [
    { ...profile, type: 'other' },
    { ...profile, keyId: '' },
    { ...profile, key: 'not-a-private-key' },
    weakProfile,
    { ...profile, issuer: 'http://issuer.example' },
    { ...profile, tokenUrl: 'https://issuer.example/oauth/v2/other' },
    { ...profile, tokenUrl: 'https://other.example/oauth/v2/token' },
    { ...profile, scopes: ['openid'] },
    { ...profile, scopes: [...profile.scopes, 'bad scope'] },
  ];
  for (const invalidProfile of invalidProfiles) {
    assert.throws(
      () => new ZitadelJwtProfileAuthProvider({
        type: 'zitadel-jwt-profile',
        profile: invalidProfile,
        fetch: async () => response(200, {}),
      }),
      (error) => error instanceof ZitadelJwtProfileAuthError
        && error.message === 'Invalid ZITADEL JWT Profile'
        && !error.message.includes(profile.key),
    );
  }

  assert.throws(
    () => new ZitadelJwtProfileAuthProvider({
      type: 'zitadel-jwt-profile',
      profile,
      assertionTtlMs: 300_001,
      fetch: async () => response(200, {}),
    }),
    /Invalid assertionTtlMs authentication setting/,
  );

  const invalidResponses = [
    {},
    { access_token: '', expires_in: 60 },
    { access_token: 'token', token_type: 'Basic', expires_in: 60 },
    { access_token: 'token', expires_in: 0 },
    { access_token: 'token', expires_in: '60' },
  ];
  for (const payload of invalidResponses) {
    const provider = new ZitadelJwtProfileAuthProvider({
      type: 'zitadel-jwt-profile',
      profile,
      fetch: async () => response(200, payload),
    });
    await assert.rejects(
      () => provider.getToken(),
      (error) => error instanceof ZitadelJwtProfileAuthError
        && error.message === 'ZITADEL token exchange returned an invalid response',
    );
  }
}

async function testOAuthClientCredentialsExchangeAndCache() {
  const profile = {
    type: 'oauth-client-credentials',
    tokenUrl: 'https://iam.example/realms/artificialflow/protocol/openid-connect/token',
    clientId: 'artificialflow-inbox-api',
    clientSecret: 'private-client-secret',
    scopes: ['openid'],
  };
  let now = 1_700_000_000_000;
  const calls = [];
  const tokens = ['oauth-one', 'oauth-two'];
  const provider = new OAuthClientCredentialsAuthProvider({
    type: 'oauth-client-credentials',
    profile,
    clock: () => now,
    refreshSkewMs: 10_000,
    fetch: async (url, init) => {
      calls.push({ url, init });
      return response(200, {
        access_token: tokens.shift(),
        token_type: 'Bearer',
        expires_in: 60,
      });
    },
  });

  assert.strictEqual(await provider.getToken(), 'oauth-one');
  assert.strictEqual(await provider.getToken(), 'oauth-one');
  assert.strictEqual(calls.length, 1);
  assert.strictEqual(calls[0].url, profile.tokenUrl);
  const form = new URLSearchParams(calls[0].init.body);
  assert.strictEqual(form.get('grant_type'), 'client_credentials');
  assert.strictEqual(form.get('client_id'), profile.clientId);
  assert.strictEqual(form.get('client_secret'), profile.clientSecret);
  assert.strictEqual(form.get('scope'), 'openid');

  now += 50_001;
  assert.strictEqual(await provider.getToken(), 'oauth-two');
  assert.strictEqual(calls.length, 2);
}

async function testOAuthClientCredentialsValidationAndClientIntegration() {
  const profile = {
    type: 'oauth-client-credentials',
    tokenUrl: 'https://iam.example/realms/artificialflow/protocol/openid-connect/token',
    clientId: 'artificialflow-inbox-api',
    clientSecret: 'private-client-secret',
  };
  assert.throws(
    () => new OAuthClientCredentialsAuthProvider({
      type: 'oauth-client-credentials',
      profile: { ...profile, tokenUrl: 'http://iam.example/token' },
      fetch: async () => response(200, {}),
    }),
    (error) => error instanceof OAuthClientCredentialsAuthError
      && error.message === 'Invalid OAuth client-credentials profile',
  );

  let rejectedBodyReads = 0;
  const rejected = new OAuthClientCredentialsAuthProvider({
    type: 'oauth-client-credentials',
    profile,
    fetch: async () => ({
      ...response(401, {}),
      text: async () => {
        rejectedBodyReads += 1;
        return JSON.stringify({ client_secret: profile.clientSecret });
      },
    }),
  });
  await assert.rejects(
    () => rejected.getToken(),
    (error) => error instanceof OAuthClientCredentialsAuthError
      && error.status === 401
      && !error.message.includes(profile.clientSecret),
  );
  assert.strictEqual(rejectedBodyReads, 0);

  const calls = [];
  const client = new ArtificialFlowClient({
    baseUrl: 'https://api.artificialflow.example.io',
    auth: { type: 'oauth-client-credentials', profile },
    fetch: async (url, init) => {
      calls.push({ url, init });
      if (url === profile.tokenUrl) {
        return response(200, {
          access_token: 'oauth-client-token',
          token_type: 'Bearer',
          expires_in: 60,
        });
      }
      return response(200, { status: 'ok' });
    },
  });
  await client.health();
  assert.strictEqual(calls.length, 2);
  assert.strictEqual(calls[1].init.headers.Authorization, 'Bearer oauth-client-token');
}

async function testClientIntegrationAndRawModeSwitch() {
  const { profile } = createProfile();
  const calls = [];
  const fetch = async (url, init) => {
    calls.push({ url, init });
    if (url === profile.tokenUrl) {
      return response(200, {
        access_token: 'profile-access-token',
        token_type: 'Bearer',
        expires_in: 60,
      });
    }
    return response(200, { status: 'ok' });
  };
  const client = new ArtificialFlowClient({
    baseUrl: 'https://api.artificialflow.example.io',
    fetch,
    auth: {
      type: 'zitadel-jwt-profile',
      profile,
    },
  });

  await client.health();
  assert.strictEqual(calls.length, 2);
  assert.strictEqual(calls[1].init.headers.Authorization, 'Bearer profile-access-token');

  client.setToken('raw-migration-token');
  await client.health();
  assert.strictEqual(calls.length, 3);
  assert.strictEqual(calls[2].init.headers.Authorization, 'Bearer raw-migration-token');

  assert.throws(
    () => new ArtificialFlowClient({
      token: 'raw-token',
      auth: { type: 'zitadel-jwt-profile', profile },
      fetch,
    }),
    /options\.token and options\.auth cannot be used together/,
  );
}

async function testWorkerReusesAccessToken() {
  const { profile } = createProfile();
  const calls = [];
  const fetch = async (url, init) => {
    calls.push({ url, init });
    if (url === profile.tokenUrl) {
      return response(200, {
        access_token: 'worker-access-token',
        expires_in: 60,
      });
    }
    if (url.endsWith('/jobs/activate')) {
      return response(200, {
        jobs: [{
          key: 'job-1',
          type: 'payment',
          processInstanceKey: 'instance-1',
          elementInstanceKey: 'element-1',
          processDefinitionKey: 'definition-1',
          elementId: 'charge',
          worker: 'worker-1',
          retries: 3,
          state: 'ACTIVATED',
          createdAt: '2026-01-01T00:00:00Z',
          updatedAt: '2026-01-01T00:00:00Z',
        }],
      });
    }
    return response(204);
  };
  const client = new ArtificialFlowClient({
    baseUrl: 'https://api.artificialflow.example.io',
    fetch,
    auth: { type: 'zitadel-jwt-profile', profile },
  });
  const worker = client.createWorker(
    'payment',
    async () => ({ paid: true }),
    { workerName: 'worker-1' },
  );

  assert.strictEqual(await worker.runOnce(), 1);
  assert.strictEqual(calls.filter((call) => call.url === profile.tokenUrl).length, 1);
  const apiCalls = calls.filter((call) => call.url !== profile.tokenUrl);
  assert.strictEqual(apiCalls.length, 2);
  assert.ok(apiCalls.every(
    (call) => call.init.headers.Authorization === 'Bearer worker-access-token',
  ));
}

async function main() {
  await testExactSignedExchangeAndCache();
  await testSingleFlightAndFailureRecovery();
  await testProfileAndResponseValidation();
  await testOAuthClientCredentialsExchangeAndCache();
  await testOAuthClientCredentialsValidationAndClientIntegration();
  await testClientIntegrationAndRawModeSwitch();
  await testWorkerReusesAccessToken();
  console.log('auth.test.js passed');
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});

/**
 * Node.js SDK worker processing examples (GitHub #36).
 *
 * Patterns:
 * - complete: activate one job and return variables (success)
 * - fail: activate one job and throw so the worker fails the job
 *
 * Usage (after npm run build in clients/nodejs-sdk):
 *   ARTIFICIALFLOW_BASE_URL=http://localhost:9100/api \
 *   ARTIFICIALFLOW_TOKEN=<bearer> \
 *   ARTIFICIALFLOW_WORKER_JOB_TYPE=golden-validate \
 *   ARTIFICIALFLOW_WORKER_MODE=complete \
 *   node examples/worker-processing.js
 */
const {
  ArtificialFlowClient,
  resolveArtificialFlowEnvironment,
} = require('../dist');

const config = resolveArtificialFlowEnvironment();
const jobType = process.env.ARTIFICIALFLOW_WORKER_JOB_TYPE || config.workerJobType;
const mode = (process.env.ARTIFICIALFLOW_WORKER_MODE || 'complete').toLowerCase();

if (!config.token && !config.zitadelProfileFile) {
  throw new Error('Set ARTIFICIALFLOW_TOKEN or ARTIFICIALFLOW_ZITADEL_PROFILE_FILE');
}
if (!jobType || String(jobType).startsWith('<')) {
  throw new Error('Set ARTIFICIALFLOW_WORKER_JOB_TYPE');
}

async function main() {
  const client = new ArtificialFlowClient({
    baseUrl: config.baseUrl,
    ...(config.zitadelProfileFile
      ? { auth: { type: 'zitadel-jwt-profile', profile: require(config.zitadelProfileFile) } }
      : { token: config.token }),
  });

  const worker = client.createWorker(jobType, async (job) => {
    console.log('activated job', job.key || job.id, 'type', job.type);
    if (mode === 'fail') {
      throw new Error('example intentional failure');
    }
    return {
      processedBy: 'nodejs-sdk-worker-example',
      handledAt: new Date().toISOString(),
      ok: true,
    };
  }, {
    workerName: process.env.ARTIFICIALFLOW_WORKER_NAME || 'nodejs-sdk-example-worker',
    autoStart: false,
    maxJobs: 1,
  });

  await worker.runOnce();
  console.log(mode === 'fail' ? 'fail path exercised' : 'complete path exercised');
}

main().catch((err) => {
  console.error(err.message || err);
  process.exit(1);
});

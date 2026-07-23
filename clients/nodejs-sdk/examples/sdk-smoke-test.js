const fs = require('node:fs');
const {
  ArtificialFlowClient,
  resolveArtificialFlowEnvironment,
} = require('../dist');

const config = resolveArtificialFlowEnvironment();

function optionalValue(value) {
  return value && !value.startsWith('<') ? value : '';
}

function loadZitadelProfile(filename) {
  try {
    return JSON.parse(fs.readFileSync(filename, 'utf8'));
  } catch {
    throw new Error('Unable to load ARTIFICIALFLOW_ZITADEL_PROFILE_FILE.');
  }
}

async function main() {
  if (Boolean(config.token) === Boolean(config.zitadelProfileFile)) {
    throw new Error('Set exactly one of ARTIFICIALFLOW_TOKEN or ARTIFICIALFLOW_ZITADEL_PROFILE_FILE.');
  }

  const auth = config.zitadelProfileFile
    ? {
      type: 'zitadel-jwt-profile',
      profile: loadZitadelProfile(config.zitadelProfileFile),
    }
    : undefined;

  const client = new ArtificialFlowClient({
    baseUrl: config.baseUrl,
    ...(auth ? { auth } : { token: config.token }),
  });

  console.log('1. Checking current authenticated principal...');
  const identity = await client.getIdentity();
  console.log(JSON.stringify(identity, null, 2));

  console.log('2. Listing workflows...');
  const workflows = await client.listWorkflows({ page: 1, pageSize: 20 });
  console.log(JSON.stringify(workflows, null, 2));

  if (optionalValue(config.workflowKey)) {
    console.log('3. Starting workflow instance...');
    const instance = await client.startInstance(config.workflowKey, {
      businessKey: config.businessKey,
      source: 'nodejs-sdk-smoke-test',
    });
    console.log(JSON.stringify(instance, null, 2));
  } else {
    console.log('3. Skipping startInstance. Set ARTIFICIALFLOW_WORKFLOW_KEY to test workflow start.');
  }

  if (optionalValue(config.messageName) && optionalValue(config.messageCorrelationKey)) {
    console.log('4. Publishing BPMN message...');
    const response = await client.publishMessage(config.messageName, config.messageCorrelationKey, {
      source: 'nodejs-sdk-smoke-test',
    });
    console.log(JSON.stringify(response, null, 2));
  } else {
    console.log('4. Skipping publishMessage. Set ARTIFICIALFLOW_MESSAGE_NAME and ARTIFICIALFLOW_MESSAGE_CORRELATION_KEY to test messages.');
  }

  if (optionalValue(config.workerJobType)) {
    console.log('5. Activating one worker job...');
    const worker = client.createWorker(config.workerJobType, async (job) => {
      console.log('Received job:');
      console.log(JSON.stringify(job, null, 2));
      return {
        handledBy: 'nodejs-sdk-smoke-test',
        handledAt: new Date().toISOString(),
      };
    }, {
      workerName: 'nodejs-sdk-smoke-worker',
      autoStart: false,
      maxJobs: 1,
    });
    const completedJobs = await worker.runOnce();
    console.log(`Completed jobs: ${completedJobs}`);
  } else {
    console.log('5. Skipping worker test. Set ARTIFICIALFLOW_WORKER_JOB_TYPE to activate and complete one job.');
  }

  console.log('SDK smoke test completed.');
}

main().catch((error) => {
  console.error('SDK smoke test failed:');
  console.error(error instanceof Error ? error.message : 'Unknown error');
  process.exitCode = 1;
});

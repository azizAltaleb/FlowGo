const { WorkflowsaClient } = require('../dist');

const config = {
  baseUrl: process.env.WORKFLOWSA_BASE_URL || 'http://localhost:9100/api',
  token: process.env.WORKFLOWSA_TOKEN || '<PASTE_WORKFLOWSA_CLIENT_ACCESS_TOKEN_HERE>',
  workflowKey: process.env.WORKFLOWSA_WORKFLOW_KEY || '<WORKFLOW_DEFINITION_KEY_OR_ID_TO_START>',
  businessKey: process.env.WORKFLOWSA_BUSINESS_KEY || `sdk-smoke-${Date.now()}`,
  messageName: process.env.WORKFLOWSA_MESSAGE_NAME || '<OPTIONAL_BPMN_MESSAGE_NAME>',
  messageCorrelationKey: process.env.WORKFLOWSA_MESSAGE_CORRELATION_KEY || '<OPTIONAL_MESSAGE_CORRELATION_KEY>',
  workerJobType: process.env.WORKFLOWSA_WORKER_JOB_TYPE || '<OPTIONAL_SERVICE_TASK_JOB_TYPE>',
};

function requireValue(name, value) {
  if (!value || value.startsWith('<')) {
    throw new Error(`Missing ${name}. Set it in this file or export ${name} in your shell.`);
  }
}

function optionalValue(value) {
  return value && !value.startsWith('<') ? value : '';
}

async function main() {
  requireValue('WORKFLOWSA_TOKEN', config.token);

  const client = new WorkflowsaClient({
    baseUrl: config.baseUrl,
    token: config.token,
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
    console.log('3. Skipping startInstance. Set WORKFLOWSA_WORKFLOW_KEY to test workflow start.');
  }

  if (optionalValue(config.messageName) && optionalValue(config.messageCorrelationKey)) {
    console.log('4. Publishing BPMN message...');
    const response = await client.publishMessage(config.messageName, config.messageCorrelationKey, {
      source: 'nodejs-sdk-smoke-test',
    });
    console.log(JSON.stringify(response, null, 2));
  } else {
    console.log('4. Skipping publishMessage. Set WORKFLOWSA_MESSAGE_NAME and WORKFLOWSA_MESSAGE_CORRELATION_KEY to test messages.');
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
    console.log('5. Skipping worker test. Set WORKFLOWSA_WORKER_JOB_TYPE to activate and complete one job.');
  }

  console.log('SDK smoke test completed.');
}

main().catch((error) => {
  console.error('SDK smoke test failed:');
  console.error(error);
  process.exitCode = 1;
});

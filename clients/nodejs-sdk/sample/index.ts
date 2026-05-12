import { FlowGoClient } from '../src';

const client = new FlowGoClient({
    baseUrl: process.env.FLOWGO_API_URL || 'http://localhost:9100/api',
    token: process.env.FLOWGO_TOKEN || '',
});

const worker = client.createWorker('payment-service', async (job) => {
    console.log(`[${new Date().toISOString()}] Processing job ${job.key}`);

    console.log(`[${new Date().toISOString()}] Job ${job.key} completed`);
    return { paymentStatus: 'success', transactionId: 'tx-123' };
}, {
    workerName: 'payment-worker',
    autoStart: true,
});

console.log('Payment Worker started. Waiting for jobs...');

process.on('SIGINT', () => {
    console.log('Stopping worker...');
    worker.stop();
    client.close();
    process.exit(0);
});

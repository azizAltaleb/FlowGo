import { ArtificialFlowClient, resolveArtificialFlowEnvironment } from '../src';

const environment = resolveArtificialFlowEnvironment();
const client = new ArtificialFlowClient({
    baseUrl: environment.baseUrl,
    token: environment.token,
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

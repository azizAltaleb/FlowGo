import { WorkflowsaClient } from './Client';
import { Job, RequestOptions } from './types';

export interface WorkerContext {
    workerName: string;
    complete: (variables?: Record<string, unknown>, options?: RequestOptions) => Promise<void>;
    fail: (errorMessage: string, retries?: number, options?: RequestOptions) => Promise<void>;
    extendLock: (lockDurationMs: number, options?: RequestOptions) => Promise<void>;
}

export type WorkerHandler = (job: Job, context: WorkerContext) => Promise<Record<string, unknown> | void>;

export interface WorkerOptions {
    maxJobs?: number;
    pollIntervalMs?: number;
    timeoutMs?: number;
    lockDurationMs?: number;
    workerName?: string;
    failRetries?: (job: Job, error: unknown) => number | undefined;
    onError?: (error: unknown) => void;
    autoStart?: boolean;
}

export class Worker {
    private readonly client: WorkflowsaClient;
    private readonly type: string;
    private readonly handler: WorkerHandler;
    private readonly options: Required<Omit<WorkerOptions, 'failRetries' | 'onError'>> & Pick<WorkerOptions, 'failRetries' | 'onError'>;
    private running = false;
    private timer?: ReturnType<typeof setTimeout>;

    constructor(client: WorkflowsaClient, type: string, handler: WorkerHandler, options: WorkerOptions = {}) {
        this.client = client;
        this.type = type;
        this.handler = handler;
        this.options = {
            maxJobs: options.maxJobs ?? 32,
            pollIntervalMs: options.pollIntervalMs ?? 1000,
            timeoutMs: options.timeoutMs ?? 5000,
            lockDurationMs: options.lockDurationMs ?? 30000,
            workerName: options.workerName || `node-worker-${process.pid}`,
            autoStart: options.autoStart ?? false,
            failRetries: options.failRetries,
            onError: options.onError,
        };
        if (this.options.autoStart) {
            this.start();
        }
    }

    public start(): void {
        if (this.running) {
            return;
        }
        this.running = true;
        void this.poll();
    }

    public stop(): void {
        this.running = false;
        if (this.timer) {
            clearTimeout(this.timer);
            this.timer = undefined;
        }
    }

    public isRunning(): boolean {
        return this.running;
    }

    public async runOnce(): Promise<number> {
        const response = await this.client.activateJobs({
            type: this.type,
            worker: this.options.workerName,
            maxJobs: this.options.maxJobs,
            timeoutMs: this.options.timeoutMs,
            lockDurationMs: this.options.lockDurationMs,
        });
        await Promise.all(response.jobs.map((job) => this.handleJob(job)));
        return response.jobs.length;
    }

    private async poll(): Promise<void> {
        if (!this.running) {
            return;
        }
        try {
            await this.runOnce();
        } catch (error) {
            if (this.options.onError) {
                this.options.onError(error);
            }
        }
        if (this.running) {
            this.timer = setTimeout(() => void this.poll(), this.options.pollIntervalMs);
        }
    }

    private async handleJob(job: Job): Promise<void> {
        const context: WorkerContext = {
            workerName: this.options.workerName,
            complete: (variables = {}, options?: RequestOptions) => this.client.completeJob(job.key, { worker: this.options.workerName, variables }, options),
            fail: (errorMessage: string, retries?: number, options?: RequestOptions) => this.client.failJob(job.key, { worker: this.options.workerName, errorMessage, retries }, options),
            extendLock: (lockDurationMs: number, options?: RequestOptions) => this.client.extendJobLock(job.key, { worker: this.options.workerName, lockDurationMs }, options),
        };
        try {
            const result = await this.handler(job, context);
            await context.complete(result || {});
        } catch (error) {
            const retries = this.options.failRetries ? this.options.failRetries(job, error) : Math.max((job.retries || 1) - 1, 0);
            await context.fail(errorMessage(error), retries);
        }
    }
}

function errorMessage(error: unknown): string {
    if (error instanceof Error) {
        return error.message;
    }
    if (typeof error === 'string') {
        return error;
    }
    return 'Job failed';
}

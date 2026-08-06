import { Worker, WorkerHandler, WorkerOptions } from './Worker';
import {
    OAuthClientCredentialsAuthProvider,
    ZitadelJwtProfileAuthProvider,
} from './Auth';
import {
    ActivateJobsRequest,
    ActivateJobsResponse,
    CreateIdentityManagementClientTokenRequest,
    CreateIdentityManagementRoleRequest,
    CreateIdentityManagementUserRequest,
    EngineMetricsResponse,
    ExtendJobLockRequest,
    FailJobRequest,
    FetchLike,
    FetchResponseLike,
    GetInboxInstanceOptions,
    GetInstanceOptions,
    HealthResponse,
    IdentityConfigResponse,
    IdentityManagementClient,
    IdentityManagementClientToken,
    IdentityManagementRole,
    IdentityManagementUser,
    IdentityResponse,
    InstanceSearchResponse,
    InboxRequestOptions,
    ListMyCompletedTransactionsOptions,
    ListOptions,
    ListUserTasksOptions,
    ListWorkflowsOptions,
    PublishEscalationRequest,
    PublishMessageRequest,
    PublishSignalRequest,
    RequestOptions,
    RotateIdentityManagementClientTokenRequest,
    SearchInstancesOptions,
    StartInstanceRequest,
    UpdateIdentityManagementRoleRequest,
    UpdateIdentityManagementUserRequest,
    UpdateVariablesRequest,
    UserTask,
    WorkerCapabilitiesResponse,
    WorkflowDefinition,
    WorkflowInstance,
    WorkflowSearchResponse,
    ArtificialFlowClientOptions,
    CompleteJobRequest,
    TokenProvider,
} from './types';

export const WorkerProtocolVersion = 'v1';
export const HeaderWorkerProtocolVersion = 'X-Workflow-Worker-Protocol-Version';
export const HeaderEngineProtocolVersion = 'X-Workflow-Engine-Protocol-Version';
export const IdempotencyKeyHeader = 'Idempotency-Key';

type RequestBody = unknown;
type AuthProvider = {
    getToken(): Promise<string>;
};

export class ArtificialFlowApiError extends Error {
    public readonly status: number;
    public readonly statusText: string;
    public readonly body: string;
    public readonly method: string;
    public readonly path: string;

    constructor(method: string, path: string, response: FetchResponseLike, body: string) {
        super(`${method} ${path} returned ${response.status} ${response.statusText}${body ? `: ${body}` : ''}`);
        this.name = 'ArtificialFlowApiError';
        this.status = response.status;
        this.statusText = response.statusText;
        this.body = body;
        this.method = method;
        this.path = path;
    }
}

export class ArtificialFlowClient {
    private baseUrl: string;
    private queryBaseUrl: string;
    private token?: TokenProvider;
    private authProvider?: AuthProvider;
    private headers?: ArtificialFlowClientOptions['headers'];
    private timeoutMs: number;
    private fetchImpl: FetchLike;

    constructor(options: ArtificialFlowClientOptions | string = {}) {
        const resolvedOptions = typeof options === 'string' ? { baseUrl: options } : options;
        if (resolvedOptions.token !== undefined && resolvedOptions.auth !== undefined) {
            throw new Error('ArtificialFlowClient options.token and options.auth cannot be used together');
        }
        this.baseUrl = normalizeBaseUrl(resolvedOptions.baseUrl || 'http://localhost:9100/api');
        this.queryBaseUrl = normalizeBaseUrl(resolvedOptions.queryBaseUrl || `${this.baseUrl}/query`);
        this.fetchImpl = resolvedOptions.fetch || defaultFetch();
        this.token = resolvedOptions.token;
        if (resolvedOptions.auth?.type === 'zitadel-jwt-profile') {
            this.authProvider = new ZitadelJwtProfileAuthProvider(
                resolvedOptions.auth,
                this.fetchImpl,
            );
        } else if (resolvedOptions.auth?.type === 'oauth-client-credentials') {
            this.authProvider = new OAuthClientCredentialsAuthProvider(
                resolvedOptions.auth,
                this.fetchImpl,
            );
        }
        this.headers = resolvedOptions.headers;
        this.timeoutMs = resolvedOptions.timeoutMs ?? 30000;
    }

    public setToken(token: string | undefined): void {
        this.authProvider = undefined;
        this.token = token;
    }

    public async health(options?: RequestOptions): Promise<HealthResponse> {
        return this.request<HealthResponse>('GET', '/health', undefined, options);
    }

    public async queryHealth(options?: RequestOptions): Promise<HealthResponse> {
        return this.queryRequest<HealthResponse>('GET', '/health', undefined, options);
    }

    public async deployWorkflow(bpmnXml: string, options?: RequestOptions): Promise<WorkflowDefinition> {
        return this.request<WorkflowDefinition>('POST', '/workflows', bpmnXml, {
            ...options,
            headers: { ...options?.headers, 'Content-Type': 'application/xml' },
        });
    }

    public async listWorkflows(options: ListWorkflowsOptions & RequestOptions = {}): Promise<WorkflowDefinition[]> {
        if (options.source === 'command') {
            return this.request<WorkflowDefinition[]>('GET', '/workflows', undefined, options);
        }
        const response = await this.searchWorkflows(options);
        return response.workflows;
    }

    public async getWorkflows(options: ListWorkflowsOptions & RequestOptions = {}): Promise<WorkflowDefinition[]> {
        return this.listWorkflows(options);
    }

    public async searchWorkflows(options: ListOptions & RequestOptions = {}): Promise<WorkflowSearchResponse> {
        return this.queryRequest<WorkflowSearchResponse>('GET', `/workflows${queryString(options, ['page', 'pageSize'])}`, undefined, options);
    }

    public async getWorkflow(id: string, options?: RequestOptions): Promise<WorkflowDefinition> {
        return this.request<WorkflowDefinition>('GET', `/workflows/${encodeURIComponent(id)}`, undefined, options);
    }

    public async deleteWorkflow(id: string, options?: RequestOptions): Promise<void> {
        await this.request<void>('DELETE', `/workflows/${encodeURIComponent(id)}`, undefined, options);
    }

    public async startInstance(workflowId: string, context?: Record<string, unknown>, options?: RequestOptions): Promise<WorkflowInstance>;
    public async startInstance(request: StartInstanceRequest, options?: RequestOptions): Promise<WorkflowInstance>;
    public async startInstance(
        workflowIdOrRequest: string | StartInstanceRequest,
        contextOrOptions: Record<string, unknown> | RequestOptions = {},
        maybeOptions: RequestOptions = {},
    ): Promise<WorkflowInstance> {
        const request = typeof workflowIdOrRequest === 'string'
            ? { workflow_id: workflowIdOrRequest, context: contextOrOptions as Record<string, unknown> }
            : workflowIdOrRequest;
        const options = typeof workflowIdOrRequest === 'string' ? maybeOptions : contextOrOptions as RequestOptions;
        return this.request<WorkflowInstance>('POST', '/instances', { ...request, context: request.context || {} }, options);
    }

    public async listActiveInstances(options?: RequestOptions): Promise<WorkflowInstance[]> {
        return this.request<WorkflowInstance[]>('GET', '/instances', undefined, options);
    }

    public async getInstances(options: SearchInstancesOptions & RequestOptions = {}): Promise<WorkflowInstance[]> {
        const response = await this.searchInstances(options);
        return response.instances;
    }

    public async searchInstances(options: SearchInstancesOptions & RequestOptions = {}): Promise<InstanceSearchResponse> {
        return this.queryRequest<InstanceSearchResponse>('GET', `/instances${queryString(options, ['workflowId', 'state', 'page', 'pageSize'])}`, undefined, options);
    }

    public async getInstance(id: string, options: GetInstanceOptions = {}): Promise<WorkflowInstance> {
        if (options.source === 'query') {
            return this.queryRequest<WorkflowInstance>('GET', `/instances/${encodeURIComponent(id)}`, undefined, options);
        }
        return this.request<WorkflowInstance>('GET', `/instances/${encodeURIComponent(id)}`, undefined, options);
    }

    public async listInboxItems(options: InboxRequestOptions): Promise<WorkflowInstance[]> {
        return this.request<WorkflowInstance[]>('GET', '/inbox', undefined, withActingUserHeaders(options));
    }

    public async listVisibleActiveInstances(options: InboxRequestOptions): Promise<WorkflowInstance[]> {
        return this.listInboxItems(options);
    }

    public async getInboxInstance(id: string, options: GetInboxInstanceOptions): Promise<WorkflowInstance> {
        return this.request<WorkflowInstance>('GET', `/inbox/instances/${encodeURIComponent(id)}${queryString(options, ['includeCompleted'])}`, undefined, withActingUserHeaders(options));
    }

    public async listUserTasks(instanceId: string, options: ListUserTasksOptions): Promise<UserTask[]> {
        const response = await this.request<{ tasks?: UserTask[] }>(
            'GET',
            `/inbox/instances/${encodeURIComponent(instanceId)}/tasks${queryString(options, ['includeCompleted'])}`,
            undefined,
            withActingUserHeaders(options),
        );
        return response.tasks || [];
    }

    public async claimUserTask(instanceId: string, executionId: string, options: InboxRequestOptions): Promise<UserTask> {
        return this.request<UserTask>(
            'POST',
            `/inbox/instances/${encodeURIComponent(instanceId)}/tasks/${encodeURIComponent(executionId)}/claim`,
            undefined,
            withActingUserHeaders(options),
        );
    }

    public async completeUserTask(instanceId: string, executionId: string, options: InboxRequestOptions): Promise<void> {
        await this.request<void>(
            'POST',
            `/inbox/instances/${encodeURIComponent(instanceId)}/tasks/${encodeURIComponent(executionId)}/complete`,
            undefined,
            withActingUserHeaders(options),
        );
    }

    public async listMyCompletedTransactions(options: ListMyCompletedTransactionsOptions): Promise<WorkflowInstance[]> {
        return this.request<WorkflowInstance[]>('GET', `/inbox/history${queryString(options, ['limit'])}`, undefined, withActingUserHeaders(options));
    }

    public async updateInstanceVariables(id: string, variables: Record<string, unknown>, options?: RequestOptions): Promise<void> {
        const request: UpdateVariablesRequest = { variables };
        await this.request<void>('POST', `/instances/${encodeURIComponent(id)}/variables`, request, options);
    }

    public async updateVariables(id: string, variables: Record<string, unknown>, options?: RequestOptions): Promise<void> {
        await this.updateInstanceVariables(id, variables, options);
    }

    public async completeTask(instanceId: string, stepId?: string, options?: RequestOptions): Promise<void> {
        await this.request<void>('POST', `/instances/${encodeURIComponent(instanceId)}/complete`, { step_id: stepId }, options);
    }

    public async deleteInstance(id: string, options?: RequestOptions): Promise<void> {
        await this.request<void>('DELETE', `/instances/${encodeURIComponent(id)}`, undefined, options);
    }

    public async publishSignal(signalName: string, payload?: Record<string, unknown>, options?: RequestOptions): Promise<void>;
    public async publishSignal(request: PublishSignalRequest, options?: RequestOptions): Promise<void>;
    public async publishSignal(
        signalNameOrRequest: string | PublishSignalRequest,
        payloadOrOptions: Record<string, unknown> | RequestOptions = {},
        maybeOptions: RequestOptions = {},
    ): Promise<void> {
        const request = typeof signalNameOrRequest === 'string'
            ? { signal_name: signalNameOrRequest, payload: payloadOrOptions as Record<string, unknown> }
            : signalNameOrRequest;
        const options = typeof signalNameOrRequest === 'string' ? maybeOptions : payloadOrOptions as RequestOptions;
        await this.request<void>('POST', '/signals', { ...request, payload: request.payload || {} }, options);
    }

    public async publishMessage(messageName: string, correlationKey: string, payload?: Record<string, unknown>, options?: RequestOptions): Promise<void>;
    public async publishMessage(request: PublishMessageRequest, options?: RequestOptions): Promise<void>;
    public async publishMessage(
        messageNameOrRequest: string | PublishMessageRequest,
        correlationKeyOrOptions: string | RequestOptions = {},
        payload: Record<string, unknown> = {},
        maybeOptions: RequestOptions = {},
    ): Promise<void> {
        const request = typeof messageNameOrRequest === 'string'
            ? { message_name: messageNameOrRequest, correlation_key: correlationKeyOrOptions as string, payload }
            : messageNameOrRequest;
        const options = typeof messageNameOrRequest === 'string' ? maybeOptions : correlationKeyOrOptions as RequestOptions;
        await this.request<void>('POST', '/messages', { ...request, payload: request.payload || {} }, options);
    }

    public async publishEscalation(escalationCode: string, payload?: Record<string, unknown>, options?: RequestOptions): Promise<void>;
    public async publishEscalation(request: PublishEscalationRequest, options?: RequestOptions): Promise<void>;
    public async publishEscalation(
        escalationCodeOrRequest: string | PublishEscalationRequest,
        payloadOrOptions: Record<string, unknown> | RequestOptions = {},
        maybeOptions: RequestOptions = {},
    ): Promise<void> {
        const request = typeof escalationCodeOrRequest === 'string'
            ? { escalation_code: escalationCodeOrRequest, payload: payloadOrOptions as Record<string, unknown> }
            : escalationCodeOrRequest;
        const options = typeof escalationCodeOrRequest === 'string' ? maybeOptions : payloadOrOptions as RequestOptions;
        await this.request<void>('POST', '/escalations', { ...request, payload: request.payload || {} }, options);
    }

    public async activateJobs(request: ActivateJobsRequest, options?: RequestOptions): Promise<ActivateJobsResponse> {
        return this.request<ActivateJobsResponse>('POST', '/jobs/activate', request, withWorkerHeaders(options));
    }

    public async getWorkerCapabilities(options?: RequestOptions): Promise<WorkerCapabilitiesResponse> {
        return this.request<WorkerCapabilitiesResponse>('GET', '/jobs/capabilities', undefined, withWorkerHeaders(options));
    }

    public async completeJob(key: string | number, request: CompleteJobRequest, options?: RequestOptions): Promise<void> {
        await this.request<void>('POST', `/jobs/${encodeURIComponent(String(key))}/complete`, request, withWorkerHeaders(options));
    }

    public async failJob(key: string | number, request: FailJobRequest, options?: RequestOptions): Promise<void> {
        await this.request<void>('POST', `/jobs/${encodeURIComponent(String(key))}/fail`, request, withWorkerHeaders(options));
    }

    public async extendJobLock(key: string | number, request: ExtendJobLockRequest, options?: RequestOptions): Promise<void> {
        await this.request<void>('POST', `/jobs/${encodeURIComponent(String(key))}/extend-lock`, request, withWorkerHeaders(options));
    }

    public async getEngineMetrics(options?: RequestOptions): Promise<EngineMetricsResponse> {
        return this.request<EngineMetricsResponse>('GET', '/internal/metrics', undefined, options);
    }

    public async getIdentity(options?: RequestOptions): Promise<IdentityResponse> {
        return this.request<IdentityResponse>('GET', '/identity/me', undefined, options);
    }

    public async getIdentityConfig(options?: RequestOptions): Promise<IdentityConfigResponse> {
        return this.request<IdentityConfigResponse>('GET', '/identity/config', undefined, options);
    }

    public async listIdentityUsers(options?: RequestOptions): Promise<IdentityManagementUser[]> {
        const response = await this.request<{ users?: IdentityManagementUser[] }>('GET', '/identity/management/users', undefined, options);
        return response.users || [];
    }

    public async getIdentityManagementUsers(options?: RequestOptions): Promise<IdentityManagementUser[]> {
        return this.listIdentityUsers(options);
    }

    public async createIdentityUser(request: CreateIdentityManagementUserRequest, options?: RequestOptions): Promise<IdentityManagementUser> {
        return this.request<IdentityManagementUser>('POST', '/identity/management/users', normalizeCreateUserRequest(request), options);
    }

    public async createIdentityManagementUser(request: CreateIdentityManagementUserRequest, options?: RequestOptions): Promise<IdentityManagementUser> {
        return this.createIdentityUser(request, options);
    }

    public async createIdentityClientToken(request: CreateIdentityManagementClientTokenRequest, options?: RequestOptions): Promise<IdentityManagementClientToken> {
        return this.request<IdentityManagementClientToken>('POST', '/identity/management/clients', normalizeCreateClientTokenRequest(request), options);
    }

    public async createIdentityManagementClientToken(request: CreateIdentityManagementClientTokenRequest, options?: RequestOptions): Promise<IdentityManagementClientToken> {
        return this.createIdentityClientToken(request, options);
    }

    public async createArtificialFlowClientToken(request: CreateIdentityManagementClientTokenRequest, options?: RequestOptions): Promise<IdentityManagementClientToken> {
        return this.createIdentityClientToken(request, options);
    }

    public async listIdentityClients(options?: RequestOptions): Promise<IdentityManagementClient[]> {
        const response = await this.request<{ clients?: IdentityManagementClient[] }>('GET', '/identity/management/clients', undefined, options);
        return response.clients || [];
    }

    public async getIdentityManagementClients(options?: RequestOptions): Promise<IdentityManagementClient[]> {
        return this.listIdentityClients(options);
    }

    public async listArtificialFlowClients(options?: RequestOptions): Promise<IdentityManagementClient[]> {
        return this.listIdentityClients(options);
    }

    public async rotateIdentityClientToken(id: string, request: RotateIdentityManagementClientTokenRequest = {}, options?: RequestOptions): Promise<IdentityManagementClientToken> {
        return this.request<IdentityManagementClientToken>('POST', `/identity/management/clients/${encodeURIComponent(id)}/tokens`, normalizeRotateClientTokenRequest(request), options);
    }

    public async rotateIdentityManagementClientToken(id: string, request: RotateIdentityManagementClientTokenRequest = {}, options?: RequestOptions): Promise<IdentityManagementClientToken> {
        return this.rotateIdentityClientToken(id, request, options);
    }

    public async revokeIdentityClientToken(id: string, tokenId: string, options?: RequestOptions): Promise<void> {
        await this.request<void>('DELETE', `/identity/management/clients/${encodeURIComponent(id)}/tokens/${encodeURIComponent(tokenId)}`, undefined, options);
    }

    public async revokeIdentityManagementClientToken(id: string, tokenId: string, options?: RequestOptions): Promise<void> {
        await this.revokeIdentityClientToken(id, tokenId, options);
    }

    public async deleteIdentityClient(id: string, options?: RequestOptions): Promise<void> {
        await this.request<void>('DELETE', `/identity/management/clients/${encodeURIComponent(id)}`, undefined, options);
    }

    public async deleteIdentityManagementClient(id: string, options?: RequestOptions): Promise<void> {
        await this.deleteIdentityClient(id, options);
    }

    public async updateIdentityUser(id: string, request: UpdateIdentityManagementUserRequest, options?: RequestOptions): Promise<IdentityManagementUser> {
        return this.request<IdentityManagementUser>('PUT', `/identity/management/users/${encodeURIComponent(id)}`, request, options);
    }

    public async updateIdentityManagementUser(id: string, request: UpdateIdentityManagementUserRequest, options?: RequestOptions): Promise<IdentityManagementUser> {
        return this.updateIdentityUser(id, request, options);
    }

    public async terminateIdentityUser(id: string, options?: RequestOptions): Promise<void> {
        await this.request<void>('POST', `/identity/management/users/${encodeURIComponent(id)}/terminate`, undefined, options);
    }

    public async terminateIdentityManagementUser(id: string, options?: RequestOptions): Promise<void> {
        await this.terminateIdentityUser(id, options);
    }

    public async reactivateIdentityUser(id: string, options?: RequestOptions): Promise<void> {
        await this.request<void>('POST', `/identity/management/users/${encodeURIComponent(id)}/reactivate`, undefined, options);
    }

    public async reactivateIdentityManagementUser(id: string, options?: RequestOptions): Promise<void> {
        await this.reactivateIdentityUser(id, options);
    }

    public async deleteIdentityUser(id: string, options?: RequestOptions): Promise<void> {
        await this.request<void>('DELETE', `/identity/management/users/${encodeURIComponent(id)}`, undefined, options);
    }

    public async deleteIdentityManagementUser(id: string, options?: RequestOptions): Promise<void> {
        await this.deleteIdentityUser(id, options);
    }

    public async listIdentityRoles(options?: RequestOptions): Promise<IdentityManagementRole[]> {
        const response = await this.request<{ roles?: IdentityManagementRole[] }>('GET', '/identity/management/roles', undefined, options);
        return response.roles || [];
    }

    public async getIdentityManagementRoles(options?: RequestOptions): Promise<IdentityManagementRole[]> {
        return this.listIdentityRoles(options);
    }

    public async createIdentityRole(request: CreateIdentityManagementRoleRequest, options?: RequestOptions): Promise<IdentityManagementRole> {
        return this.request<IdentityManagementRole>('POST', '/identity/management/roles', { ...request, group: request.group || 'ArtificialFlow' }, options);
    }

    public async createIdentityManagementRole(request: CreateIdentityManagementRoleRequest, options?: RequestOptions): Promise<IdentityManagementRole> {
        return this.createIdentityRole(request, options);
    }

    public async updateIdentityRole(key: string, request: UpdateIdentityManagementRoleRequest, options?: RequestOptions): Promise<IdentityManagementRole> {
        return this.request<IdentityManagementRole>('PUT', `/identity/management/roles/${encodeURIComponent(key)}`, request, options);
    }

    public async updateIdentityManagementRole(key: string, request: UpdateIdentityManagementRoleRequest, options?: RequestOptions): Promise<IdentityManagementRole> {
        return this.updateIdentityRole(key, request, options);
    }

    public async deleteIdentityRole(key: string, options?: RequestOptions): Promise<void> {
        await this.request<void>('DELETE', `/identity/management/roles/${encodeURIComponent(key)}`, undefined, options);
    }

    public async deleteIdentityManagementRole(key: string, options?: RequestOptions): Promise<void> {
        await this.deleteIdentityRole(key, options);
    }

    public createWorker(type: string, handler: WorkerHandler, options: WorkerOptions = {}): Worker {
        return new Worker(this, type, handler, options);
    }

    public close(): void {}

    private request<T>(method: string, path: string, body?: RequestBody, options: RequestOptions = {}): Promise<T> {
        return this.doRequest<T>(this.baseUrl, method, path, body, options);
    }

    private queryRequest<T>(method: string, path: string, body?: RequestBody, options: RequestOptions = {}): Promise<T> {
        return this.doRequest<T>(this.queryBaseUrl, method, path, body, options);
    }

    private async doRequest<T>(baseUrl: string, method: string, path: string, body?: RequestBody, options: RequestOptions = {}): Promise<T> {
        const controller = options.signal ? undefined : new AbortController();
        const timeout = controller ? setTimeout(() => controller.abort(), this.timeoutMs) : undefined;
        try {
            const headers = await this.resolveHeaders(options);
            const requestBody = typeof body === 'string' ? body : body === undefined ? undefined : JSON.stringify(body);
            if (requestBody !== undefined && !headers['Content-Type']) {
                headers['Content-Type'] = 'application/json';
            }
            const response = await this.fetchImpl(`${baseUrl}${path}`, {
                method,
                headers,
                body: requestBody,
                signal: options.signal || controller?.signal,
            });
            const responseText = await response.text();
            if (!response.ok) {
                throw new ArtificialFlowApiError(method, path, response, responseText);
            }
            if (response.status === 204 || responseText.trim() === '') {
                return undefined as T;
            }
            const contentType = response.headers.get('content-type') || '';
            if (contentType.includes('application/json')) {
                return JSON.parse(responseText) as T;
            }
            return responseText as T;
        } finally {
            if (timeout) {
                clearTimeout(timeout);
            }
        }
    }

    private async resolveHeaders(options: RequestOptions): Promise<Record<string, string>> {
        const headers: Record<string, string> = {
            Accept: 'application/json',
            'X-Correlation-ID': options.correlationId || generateCorrelationId(),
        };
        const extraHeaders = await resolveMaybe(this.headers);
        Object.assign(headers, extraHeaders || {});
        const token = this.authProvider
            ? await this.authProvider.getToken()
            : await resolveMaybe(this.token);
        if (token) {
            headers.Authorization = `Bearer ${token}`;
        }
        if (options.idempotencyKey) {
            headers[IdempotencyKeyHeader] = options.idempotencyKey;
        }
        Object.assign(headers, options.headers || {});
        return headers;
    }
}

function normalizeBaseUrl(value: string): string {
    return value.replace(/\/+$/, '');
}

function defaultFetch(): FetchLike {
    const fetchImpl = globalThis.fetch;
    if (!fetchImpl) {
        throw new Error('No fetch implementation available. Use Node.js 18+ or pass options.fetch.');
    }
    return fetchImpl as unknown as FetchLike;
}

async function resolveMaybe<T>(value: T | (() => T | Promise<T>) | undefined): Promise<T | undefined> {
    if (typeof value === 'function') {
        return (value as () => T | Promise<T>)();
    }
    return value;
}

function generateCorrelationId(): string {
    return `sdk-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

function queryString(options: object, keys: string[]): string {
    const params = new URLSearchParams();
    const source = options as Record<string, unknown>;
    for (const key of keys) {
        const value = source[key];
        if (value !== undefined && value !== null && value !== '') {
            params.set(String(key), String(value));
        }
    }
    const encoded = params.toString();
    return encoded ? `?${encoded}` : '';
}

function withWorkerHeaders(options: RequestOptions = {}): RequestOptions {
    return {
        ...options,
        headers: {
            [HeaderWorkerProtocolVersion]: WorkerProtocolVersion,
            ...options.headers,
        },
    };
}

function withActingUserHeaders(options: InboxRequestOptions): RequestOptions {
    const actingUser = options.actingUser;
    if (!actingUser || (!actingUser.subject && !actingUser.username && !actingUser.email)) {
        throw new Error('actingUser.subject, actingUser.username, or actingUser.email is required for inbox requests');
    }

    const headers: Record<string, string> = { ...options.headers };
    if (actingUser.subject) {
        headers['X-ArtificialFlow-Acting-Subject'] = actingUser.subject;
    }
    if (actingUser.username) {
        headers['X-ArtificialFlow-Acting-Username'] = actingUser.username;
    }
    if (actingUser.email) {
        headers['X-ArtificialFlow-Acting-Email'] = actingUser.email;
    }
    if (actingUser.name) {
        headers['X-ArtificialFlow-Acting-Name'] = actingUser.name;
    }
    if (actingUser.roles && actingUser.roles.length > 0) {
        headers['X-ArtificialFlow-Acting-Roles'] = actingUser.roles.join(',');
    }

    return {
        correlationId: options.correlationId,
        idempotencyKey: options.idempotencyKey,
        signal: options.signal,
        headers,
    };
}

function normalizeCreateUserRequest(request: CreateIdentityManagementUserRequest): CreateIdentityManagementUserRequest {
    return {
        ...request,
        username: request.username || '',
        password_change_required: request.password_change_required ?? false,
        roles: request.roles || [],
    };
}

function normalizeCreateClientTokenRequest(request: CreateIdentityManagementClientTokenRequest): CreateIdentityManagementClientTokenRequest {
    return {
        ...request,
        username: request.username || '',
        description: request.description || '',
        environment: request.environment || '',
        owner_email: request.owner_email || '',
        purpose: request.purpose || '',
        token_expires_at: request.token_expires_at || '',
    };
}

function normalizeRotateClientTokenRequest(request: RotateIdentityManagementClientTokenRequest): RotateIdentityManagementClientTokenRequest {
    return {
        ...request,
        token_expires_at: request.token_expires_at || '',
    };
}


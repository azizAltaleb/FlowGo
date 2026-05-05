export type JsonPrimitive = string | number | boolean | null;
export type JsonValue = JsonPrimitive | JsonObject | JsonArray;
export interface JsonObject {
    [key: string]: JsonValue;
}
export interface JsonArray extends Array<JsonValue> {}

export interface WorkflowsaClientOptions {
    baseUrl?: string;
    queryBaseUrl?: string;
    token?: string | (() => string | Promise<string>);
    headers?: Record<string, string> | (() => Record<string, string> | Promise<Record<string, string>>);
    timeoutMs?: number;
    fetch?: FetchLike;
}

export interface RequestOptions {
    correlationId?: string;
    idempotencyKey?: string;
    headers?: Record<string, string>;
    signal?: AbortSignal;
}

export interface FetchResponseLike {
    ok: boolean;
    status: number;
    statusText: string;
    headers: {
        get(name: string): string | null;
    };
    text(): Promise<string>;
}

export type FetchLike = (input: string, init?: {
    method?: string;
    headers?: Record<string, string>;
    body?: string;
    signal?: AbortSignal;
}) => Promise<FetchResponseLike>;

export interface HealthResponse {
    status: string;
    [key: string]: JsonValue;
}

export interface WorkflowDefinition {
    id: string;
    process_definition_id: string;
    name: string;
    version: number;
    resource_name: string;
    deployment_id: string;
    tenant_id: string;
    resource_checksum: string;
    bpmn_xml?: string;
    created_at: string;
    steps?: unknown[];
}

export interface WorkflowSearchResponse {
    workflows: WorkflowDefinition[];
    total: number;
}

export interface Execution {
    id?: string;
    step_id?: string;
    status?: string;
    start_time?: string;
    [key: string]: unknown;
}

export interface WorkflowInstance {
    id: string;
    workflow_id: string;
    status: string;
    parent_instance_id?: string;
    parent_execution_id?: string;
    context: Record<string, unknown>;
    executions?: Execution[];
    created_at: string;
    updated_at: string;
}

export interface InstanceSearchResponse {
    instances: WorkflowInstance[];
    total: number;
}

export interface ListOptions {
    page?: number;
    pageSize?: number;
}

export interface ListWorkflowsOptions extends ListOptions {
    source?: 'command' | 'query';
}

export interface SearchInstancesOptions extends ListOptions {
    workflowId?: string;
    state?: string;
}

export interface GetInstanceOptions extends RequestOptions {
    source?: 'command' | 'query';
}

export interface StartInstanceRequest {
    workflow_id: string;
    context?: Record<string, unknown>;
}

export interface UpdateVariablesRequest {
    variables: Record<string, unknown>;
}

export interface CompleteTaskRequest {
    step_id?: string;
}

export interface PublishSignalRequest {
    signal_name: string;
    payload?: Record<string, unknown>;
}

export interface PublishMessageRequest {
    message_name: string;
    correlation_key: string;
    payload?: Record<string, unknown>;
}

export interface Job {
    key: string;
    type: string;
    processInstanceKey: string;
    elementInstanceKey: string;
    processDefinitionKey: string;
    elementId: string;
    worker: string;
    retries: number;
    state: string;
    lockExpirationTime?: string | null;
    createdAt: string;
    updatedAt: string;
}

export interface ActivateJobsRequest {
    type: string;
    worker: string;
    maxJobs?: number;
    timeoutMs?: number;
    lockDurationMs?: number;
}

export interface ActivateJobsResponse {
    jobs: Job[];
}

export interface CompleteJobRequest {
    worker: string;
    variables?: Record<string, unknown>;
}

export interface FailJobRequest {
    worker: string;
    errorMessage: string;
    retries?: number;
}

export interface ExtendJobLockRequest {
    worker: string;
    lockDurationMs: number;
}

export interface WorkerCapabilitiesResponse {
    protocolVersion: string;
    capabilities: string[];
}

export interface EngineMetricsResponse {
    outboxPending: number;
    outboxPublishSuccess: number;
    outboxPublishFailure: number;
    outboxPublishLagSec: number;
    outboxMaxAttempts: number;
    idempotencyHit: number;
    idempotencyMiss: number;
}

export interface IdentityPrincipal {
    subject: string;
    issuer?: string;
    audience?: string[];
    email?: string;
    name?: string;
    tenant_id?: string;
    roles?: string[];
    scopes?: string[];
    token_mode?: string;
    claims?: Record<string, unknown>;
}

export interface IdentityResponse {
    authenticated: boolean;
    principal?: IdentityPrincipal;
}

export interface IdentityConfigResponse {
    deployment_mode: string;
    configuration_source: string;
    provider_name: string;
    auth_enabled: boolean;
    frontend_auth_enabled: boolean;
    frontend_oidc_authority: string;
    frontend_oidc_client_id: string;
    token_validation_mode: string;
    internal_issuer_url: string;
    external_issuer_url: string;
    client_id: string;
    introspection_url: string;
    introspection_client_id: string;
    introspection_auth_method: string;
    enforce_audience: boolean;
    allow_insecure_issuer: boolean;
    claim_subject_path: string;
    claim_roles_path: string;
    claim_scopes_path: string;
    claim_tenant_path: string;
    claim_email_path: string;
    claim_name_path: string;
    standard_roles: string[];
}

export interface IdentityManagementUser {
    id: string;
    username: string;
    preferred_login_name: string;
    display_name: string;
    given_name: string;
    family_name: string;
    email: string;
    email_verified: boolean;
    state: string;
    type: string;
    created_at: string;
    changed_at: string;
    roles: string[];
}

export interface IdentityManagementRole {
    key: string;
    display_name: string;
    group: string;
}

export interface IdentityManagementClientToken {
    client_id: string;
    username: string;
    name: string;
    description: string;
    environment: string;
    owner_email: string;
    purpose: string;
    role: string;
    token_id: string;
    token: string;
    token_created_at: string;
    token_expires_at: string;
}

export interface IdentityManagementClientTokenSummary {
    token_id: string;
    token_created_at: string;
    token_changed_at: string;
    token_expires_at: string;
    status: string;
}

export interface IdentityManagementClient {
    client_id: string;
    username: string;
    name: string;
    description: string;
    environment: string;
    owner_email: string;
    purpose: string;
    role: string;
    state: string;
    created_at: string;
    changed_at: string;
    tokens: IdentityManagementClientTokenSummary[];
}

export interface CreateIdentityManagementClientTokenRequest {
    username?: string;
    name: string;
    description?: string;
    environment?: string;
    owner_email?: string;
    purpose?: string;
    token_expires_at?: string;
}

export interface RotateIdentityManagementClientTokenRequest {
    token_expires_at?: string;
}

export interface CreateIdentityManagementUserRequest {
    username?: string;
    given_name: string;
    family_name: string;
    email: string;
    password: string;
    password_change_required?: boolean;
    roles?: string[];
}

export interface UpdateIdentityManagementUserRequest {
    username?: string;
    given_name?: string;
    family_name?: string;
    display_name?: string;
    email?: string;
    roles?: string[];
}

export interface CreateIdentityManagementRoleRequest {
    key: string;
    display_name: string;
    group?: string;
}

export interface UpdateIdentityManagementRoleRequest {
    display_name: string;
    group?: string;
}

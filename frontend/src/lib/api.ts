import { createLogger, generateCorrelationId } from './logger';
import { runtimeConfig } from './runtimeConfig';

const API_BASE_URL = (runtimeConfig.apiUrl || "/api").replace(/\/+$/, "");
const log = createLogger('api');

let accessToken: string | null = null;

export const setAccessToken = (token: string | null) => {
  accessToken = token;
};

const getHeaders = (correlationId: string) => {
  const headers: Record<string, string> = {
    'X-Correlation-ID': correlationId,
  };
  if (accessToken) {
    headers['Authorization'] = `Bearer ${accessToken}`;
  }
  return headers;
};

export interface WorkflowDefinition {
  id: string;
  process_definition_id: string;
  name: string;
  version: number;
  resource_name: string;
  deployment_id: string;
  tenant_id: string;
  resource_checksum: string;
  bpmn_xml: string;
  created_at: string;
}

export interface WorkflowInstance {
  id: string;
  workflow_id: string;
  status: "PENDING" | "RUNNING" | "COMPLETED" | "FAILED";
  current_step?: string;
  parent_instance_id?: string;
  parent_execution_id?: string;
  context: Record<string, unknown>;
  executions: Execution[];
  created_at: string;
  updated_at: string;
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

export interface CreateIdentityManagementUserRequest {
  username: string;
  given_name: string;
  family_name: string;
  email: string;
  password: string;
  password_change_required: boolean;
  roles: string[];
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
  group: string;
}

export interface UpdateIdentityManagementRoleRequest {
  display_name: string;
  group: string;
}

export interface Execution {
  id: string;
  step_id: string;
  status: string;
  start_time: string;
}

export const api = {
  // Workflows
  getWorkflows: async (): Promise<WorkflowDefinition[]> => {
    const correlationId = generateCorrelationId();
    const endTimer = log.time('getWorkflows');
    log.debug('fetching workflows', { correlationId });

    // CQRS: Read from Query Service
    const response = await fetch(`${API_BASE_URL}/query/workflows?page=1&pageSize=100`, {
      headers: getHeaders(correlationId),
    });
    if (!response.ok) {
      log.error('failed to fetch workflows', { status: response.status, statusText: response.statusText });
      throw new Error(`Failed to fetch workflows: ${response.statusText}`);
    }
    const data = await response.json();
    log.info('workflows fetched', { count: data.workflows?.length || 0 });
    endTimer();
    
    return (data.workflows || []).map((w: Record<string, unknown>) => {
        if (w.id && w.process_definition_id) return w as unknown as WorkflowDefinition;

        return {
            id: String(w.key),
            process_definition_id: w.bpmnProcessId,
            name: w.bpmnProcessId, // Use ID as name if not separate
            version: w.version,
            resource_name: w.resourceName,
            deployment_id: String(w.deploymentKey),
            tenant_id: w.tenantId,
            resource_checksum: w.resourceChecksum,
            bpmn_xml: w.resource, // Resource is bytes/string
            created_at: w.createdAt
        } as WorkflowDefinition;
    });
  },

  getWorkflow: async (id: string): Promise<WorkflowDefinition> => {
    const correlationId = generateCorrelationId();
    log.debug('fetching workflow', { workflowId: id, correlationId });

    const response = await fetch(`${API_BASE_URL}/workflows/${id}`, {
      headers: getHeaders(correlationId),
    });
    if (!response.ok) {
      log.error('failed to fetch workflow', { workflowId: id, status: response.status });
      throw new Error(`Failed to fetch workflow ${id}: ${response.statusText}`);
    }
    log.debug('workflow fetched', { workflowId: id });
    return response.json();
  },

  deployWorkflow: async (bpmnXml: string): Promise<WorkflowDefinition> => {
    const correlationId = generateCorrelationId();
    const endTimer = log.time('deployWorkflow');
    log.info('deploying workflow', { xmlSize: bpmnXml.length, correlationId });

    const headers = getHeaders(correlationId);
    headers["Content-Type"] = "application/xml";

    const response = await fetch(`${API_BASE_URL}/workflows`, {
      method: "POST",
      headers: headers,
      body: bpmnXml,
    });
    if (!response.ok) {
      log.error('failed to deploy workflow', { status: response.status, statusText: response.statusText });
      throw new Error(`Failed to deploy workflow: ${response.statusText}`);
    }
    const result = await response.json();
    log.info('workflow deployed', { workflowId: result.id, workflowName: result.name });
    endTimer();
    return result;
  },

  deleteWorkflow: async (id: string): Promise<void> => {
    const correlationId = generateCorrelationId();
    log.info('deleting workflow', { workflowId: id, correlationId });

    const response = await fetch(`${API_BASE_URL}/workflows/${id}`, {
      method: "DELETE",
      headers: getHeaders(correlationId),
    });
    if (!response.ok) {
      log.error('failed to delete workflow', { workflowId: id, status: response.status });
      throw new Error(`Failed to delete workflow: ${response.statusText}`);
    }
    log.info('workflow deleted', { workflowId: id });
  },

  // Instances
  getInstances: async (): Promise<WorkflowInstance[]> => {
    const correlationId = generateCorrelationId();
    const endTimer = log.time('getInstances');
    log.debug('fetching instances', { correlationId });

    // CQRS: Read from Query Service
    const response = await fetch(`${API_BASE_URL}/query/instances?page=1&pageSize=100`, {
      headers: getHeaders(correlationId),
    });
    if (!response.ok) {
      log.error('failed to fetch instances', { status: response.status, statusText: response.statusText });
      throw new Error(`Failed to fetch instances: ${response.statusText}`);
    }
    const data = await response.json();
    log.info('instances fetched', { count: data.instances?.length || 0, total: data.total });
    endTimer();
    
    // Map Query Service (Engine-like) response to Frontend (Legacy) model
    return (data.instances || []).map((i: Record<string, unknown>) => {
        // If it already looks like the legacy model, return it (id and status present)
        if (i.id && i.status) return i as unknown as WorkflowInstance;

        // Map Engine model (key/state) to Legacy model (id/status)
        let status: "PENDING" | "RUNNING" | "COMPLETED" | "FAILED" = "PENDING";
        if (i.state === "ACTIVE" || i.state === "ACTIVATED") status = "RUNNING";
        else if (i.state === "COMPLETED") status = "COMPLETED";
        else if (i.state === "CANCELED" || i.state === "TERMINATED") status = "FAILED";

        return {
            id: String(i.key),
            workflow_id: String(i.processDefinitionKey),
            status: status,
            current_step: "", // Not available in summary list
            context: i.context || {},
            executions: [], // Not available in summary list
            created_at: i.createdAt,
            updated_at: i.endTime || i.createdAt // Use endTime if available, else createdAt
        } as unknown as WorkflowInstance;
    });
  },

  startInstance: async (workflowId: string, context: Record<string, unknown> = {}): Promise<WorkflowInstance> => {
    const correlationId = generateCorrelationId();
    const endTimer = log.time('startInstance');
    log.info('starting instance', { workflowId, contextKeys: Object.keys(context), correlationId });

    const headers = getHeaders(correlationId);
    headers["Content-Type"] = "application/json";

    const response = await fetch(`${API_BASE_URL}/instances`, {
      method: "POST",
      headers: headers,
      body: JSON.stringify({
        workflow_id: workflowId,
        context: context,
      }),
    });
    if (!response.ok) {
      log.error('failed to start instance', { workflowId, status: response.status });
      throw new Error(`Failed to start instance: ${response.statusText}`);
    }
    const result = await response.json();
    log.info('instance started', { instanceId: result.id, workflowId });
    endTimer();
    return result;
  },

  getInstance: async (id: string): Promise<WorkflowInstance> => {
    const correlationId = generateCorrelationId();
    log.debug('fetching instance', { instanceId: id, correlationId });

    const response = await fetch(`${API_BASE_URL}/instances/${id}`, {
      headers: getHeaders(correlationId),
    });
    if (!response.ok) {
      log.error('failed to fetch instance', { instanceId: id, status: response.status });
      throw new Error(`Failed to fetch instance ${id}: ${response.statusText}`);
    }
    log.debug('instance fetched', { instanceId: id });
    return response.json();
  },

  updateInstanceVariables: async (id: string, variables: Record<string, unknown>): Promise<void> => {
    const correlationId = generateCorrelationId();
    log.info('updating instance variables', { instanceId: id, variableNames: Object.keys(variables), correlationId });

    const headers = getHeaders(correlationId);
    headers["Content-Type"] = "application/json";

    const response = await fetch(`${API_BASE_URL}/instances/${id}/variables`, {
      method: "POST",
      headers: headers,
      body: JSON.stringify({ variables }),
    });
    if (!response.ok) {
      log.error('failed to update instance variables', { instanceId: id, status: response.status });
      throw new Error(`Failed to update instance variables: ${response.statusText}`);
    }
    log.info('instance variables updated', { instanceId: id, variableCount: Object.keys(variables).length });
  },

  completeTask: async (instanceId: string, stepId?: string): Promise<void> => {
    const correlationId = generateCorrelationId();
    log.info('completing task', { instanceId, stepId, correlationId });

    const headers = getHeaders(correlationId);
    headers["Content-Type"] = "application/json";

    const response = await fetch(`${API_BASE_URL}/instances/${instanceId}/complete`, {
      method: "POST",
      headers: headers,
      body: JSON.stringify({ step_id: stepId }),
    });
    if (!response.ok) {
      log.error('failed to complete task', { instanceId, stepId, status: response.status });
      throw new Error(`Failed to complete task: ${response.statusText}`);
    }
    log.info('task completed', { instanceId, stepId });
  },

  deleteInstance: async (id: string): Promise<void> => {
    const correlationId = generateCorrelationId();
    log.info('deleting instance', { instanceId: id, correlationId });

    const response = await fetch(`${API_BASE_URL}/instances/${id}`, {
      method: "DELETE",
      headers: getHeaders(correlationId),
    });
    if (!response.ok) {
      log.error('failed to delete instance', { instanceId: id, status: response.status });
      throw new Error(`Failed to delete instance: ${response.statusText}`);
    }
    log.info('instance deleted', { instanceId: id });
  },

  getIdentity: async (): Promise<IdentityResponse> => {
    const correlationId = generateCorrelationId();
    log.debug('fetching identity', { correlationId });

    const response = await fetch(`${API_BASE_URL}/identity/me`, {
      headers: getHeaders(correlationId),
    });
    if (!response.ok) {
      log.error('failed to fetch identity', { correlationId, status: response.status });
      throw new Error(`Failed to fetch identity: ${response.statusText}`);
    }
    return response.json();
  },

  getIdentityConfig: async (): Promise<IdentityConfigResponse> => {
    const correlationId = generateCorrelationId();
    log.debug('fetching identity config', { correlationId });

    const response = await fetch(`${API_BASE_URL}/identity/config`, {
      headers: getHeaders(correlationId),
    });
    if (!response.ok) {
      log.error('failed to fetch identity config', { correlationId, status: response.status });
      throw new Error(`Failed to fetch identity config: ${response.statusText}`);
    }
    return response.json();
  },

  getIdentityManagementUsers: async (): Promise<IdentityManagementUser[]> => {
    const correlationId = generateCorrelationId();
    const response = await fetch(`${API_BASE_URL}/identity/management/users`, {
      headers: getHeaders(correlationId),
    });
    if (!response.ok) {
      throw new Error(`Failed to fetch identity users: ${response.statusText}`);
    }
    const data = await response.json();
    return data.users || [];
  },

  createIdentityManagementUser: async (request: CreateIdentityManagementUserRequest): Promise<IdentityManagementUser> => {
    const correlationId = generateCorrelationId();
    const headers = getHeaders(correlationId);
    headers["Content-Type"] = "application/json";
    const response = await fetch(`${API_BASE_URL}/identity/management/users`, {
      method: "POST",
      headers,
      body: JSON.stringify(request),
    });
    if (!response.ok) {
      throw new Error(`Failed to create identity user: ${response.statusText}`);
    }
    return response.json();
  },

  createIdentityManagementClientToken: async (request: CreateIdentityManagementClientTokenRequest): Promise<IdentityManagementClientToken> => {
    const correlationId = generateCorrelationId();
    const headers = getHeaders(correlationId);
    headers["Content-Type"] = "application/json";
    const response = await fetch(`${API_BASE_URL}/identity/management/clients`, {
      method: "POST",
      headers,
      body: JSON.stringify(request),
    });
    if (!response.ok) {
      throw new Error(`Failed to create GoFlow client token: ${response.statusText}`);
    }
    return response.json();
  },

  getIdentityManagementClients: async (): Promise<IdentityManagementClient[]> => {
    const correlationId = generateCorrelationId();
    const response = await fetch(`${API_BASE_URL}/identity/management/clients`, {
      headers: getHeaders(correlationId),
    });
    if (!response.ok) {
      throw new Error(`Failed to fetch GoFlow clients: ${response.statusText}`);
    }
    const data = await response.json();
    return data.clients || [];
  },

  rotateIdentityManagementClientToken: async (id: string, request: RotateIdentityManagementClientTokenRequest): Promise<IdentityManagementClientToken> => {
    const correlationId = generateCorrelationId();
    const headers = getHeaders(correlationId);
    headers["Content-Type"] = "application/json";
    const response = await fetch(`${API_BASE_URL}/identity/management/clients/${encodeURIComponent(id)}/tokens`, {
      method: "POST",
      headers,
      body: JSON.stringify(request),
    });
    if (!response.ok) {
      throw new Error(`Failed to rotate GoFlow client token: ${response.statusText}`);
    }
    return response.json();
  },

  revokeIdentityManagementClientToken: async (id: string, tokenId: string): Promise<void> => {
    const correlationId = generateCorrelationId();
    const response = await fetch(`${API_BASE_URL}/identity/management/clients/${encodeURIComponent(id)}/tokens/${encodeURIComponent(tokenId)}`, {
      method: "DELETE",
      headers: getHeaders(correlationId),
    });
    if (!response.ok) {
      throw new Error(`Failed to revoke GoFlow client token: ${response.statusText}`);
    }
  },

  deleteIdentityManagementClient: async (id: string): Promise<void> => {
    const correlationId = generateCorrelationId();
    const response = await fetch(`${API_BASE_URL}/identity/management/clients/${encodeURIComponent(id)}`, {
      method: "DELETE",
      headers: getHeaders(correlationId),
    });
    if (!response.ok) {
      throw new Error(`Failed to delete GoFlow client: ${response.statusText}`);
    }
  },

  updateIdentityManagementUser: async (id: string, request: UpdateIdentityManagementUserRequest): Promise<IdentityManagementUser> => {
    const correlationId = generateCorrelationId();
    const headers = getHeaders(correlationId);
    headers["Content-Type"] = "application/json";
    const response = await fetch(`${API_BASE_URL}/identity/management/users/${encodeURIComponent(id)}`, {
      method: "PUT",
      headers,
      body: JSON.stringify(request),
    });
    if (!response.ok) {
      throw new Error(`Failed to update identity user: ${response.statusText}`);
    }
    return response.json();
  },

  terminateIdentityManagementUser: async (id: string): Promise<void> => {
    const correlationId = generateCorrelationId();
    const response = await fetch(`${API_BASE_URL}/identity/management/users/${encodeURIComponent(id)}/terminate`, {
      method: "POST",
      headers: getHeaders(correlationId),
    });
    if (!response.ok) {
      throw new Error(`Failed to terminate identity user: ${response.statusText}`);
    }
  },

  reactivateIdentityManagementUser: async (id: string): Promise<void> => {
    const correlationId = generateCorrelationId();
    const response = await fetch(`${API_BASE_URL}/identity/management/users/${encodeURIComponent(id)}/reactivate`, {
      method: "POST",
      headers: getHeaders(correlationId),
    });
    if (!response.ok) {
      throw new Error(`Failed to reactivate identity user: ${response.statusText}`);
    }
  },

  deleteIdentityManagementUser: async (id: string): Promise<void> => {
    const correlationId = generateCorrelationId();
    const response = await fetch(`${API_BASE_URL}/identity/management/users/${encodeURIComponent(id)}`, {
      method: "DELETE",
      headers: getHeaders(correlationId),
    });
    if (!response.ok) {
      throw new Error(`Failed to delete identity user: ${response.statusText}`);
    }
  },

  getIdentityManagementRoles: async (): Promise<IdentityManagementRole[]> => {
    const correlationId = generateCorrelationId();
    const response = await fetch(`${API_BASE_URL}/identity/management/roles`, {
      headers: getHeaders(correlationId),
    });
    if (!response.ok) {
      throw new Error(`Failed to fetch identity roles: ${response.statusText}`);
    }
    const data = await response.json();
    return data.roles || [];
  },

  createIdentityManagementRole: async (request: CreateIdentityManagementRoleRequest): Promise<IdentityManagementRole> => {
    const correlationId = generateCorrelationId();
    const headers = getHeaders(correlationId);
    headers["Content-Type"] = "application/json";
    const response = await fetch(`${API_BASE_URL}/identity/management/roles`, {
      method: "POST",
      headers,
      body: JSON.stringify(request),
    });
    if (!response.ok) {
      throw new Error(`Failed to create identity role: ${response.statusText}`);
    }
    return response.json();
  },

  updateIdentityManagementRole: async (key: string, request: UpdateIdentityManagementRoleRequest): Promise<IdentityManagementRole> => {
    const correlationId = generateCorrelationId();
    const headers = getHeaders(correlationId);
    headers["Content-Type"] = "application/json";
    const response = await fetch(`${API_BASE_URL}/identity/management/roles/${encodeURIComponent(key)}`, {
      method: "PUT",
      headers,
      body: JSON.stringify(request),
    });
    if (!response.ok) {
      throw new Error(`Failed to update identity role: ${response.statusText}`);
    }
    return response.json();
  },

  deleteIdentityManagementRole: async (key: string): Promise<void> => {
    const correlationId = generateCorrelationId();
    const response = await fetch(`${API_BASE_URL}/identity/management/roles/${encodeURIComponent(key)}`, {
      method: "DELETE",
      headers: getHeaders(correlationId),
    });
    if (!response.ok) {
      throw new Error(`Failed to delete identity role: ${response.statusText}`);
    }
  },
};

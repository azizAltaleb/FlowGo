export type ArtificialFlowEnvironmentSource = Record<string, string | undefined>;

export interface ArtificialFlowEnvironmentConfig {
    baseUrl: string;
    token: string;
    zitadelProfileFile: string;
    workflowKey: string;
    businessKey: string;
    messageName: string;
    messageCorrelationKey: string;
    workerJobType: string;
}

export function readArtificialFlowEnvironmentValue(
    environment: ArtificialFlowEnvironmentSource,
    canonicalName: string,
    legacyName: string,
    fallback = '',
): string {
    return environment[canonicalName]?.trim()
        || environment[legacyName]?.trim()
        || fallback;
}

export function resolveArtificialFlowEnvironment(
    environment: ArtificialFlowEnvironmentSource =
        typeof process === 'undefined' ? {} : process.env,
    now: () => number = Date.now,
): ArtificialFlowEnvironmentConfig {
    const read = (canonicalName: string, legacyName: string, fallback = '') =>
        readArtificialFlowEnvironmentValue(environment, canonicalName, legacyName, fallback);

    return {
        baseUrl:
            environment.ARTIFICIALFLOW_BASE_URL?.trim()
            || environment.ARTIFICIALFLOW_API_URL?.trim()
            || environment.FLOWGO_BASE_URL?.trim()
            || environment.FLOWGO_API_URL?.trim()
            || 'http://localhost:9100/api',
        token: read('ARTIFICIALFLOW_TOKEN', 'FLOWGO_TOKEN'),
        zitadelProfileFile: read('ARTIFICIALFLOW_ZITADEL_PROFILE_FILE', 'FLOWGO_ZITADEL_PROFILE_FILE'),
        workflowKey: read(
            'ARTIFICIALFLOW_WORKFLOW_KEY',
            'FLOWGO_WORKFLOW_KEY',
            '<WORKFLOW_DEFINITION_KEY_OR_ID_TO_START>',
        ),
        businessKey: read(
            'ARTIFICIALFLOW_BUSINESS_KEY',
            'FLOWGO_BUSINESS_KEY',
            `sdk-smoke-${now()}`,
        ),
        messageName: read(
            'ARTIFICIALFLOW_MESSAGE_NAME',
            'FLOWGO_MESSAGE_NAME',
            '<OPTIONAL_BPMN_MESSAGE_NAME>',
        ),
        messageCorrelationKey: read(
            'ARTIFICIALFLOW_MESSAGE_CORRELATION_KEY',
            'FLOWGO_MESSAGE_CORRELATION_KEY',
            '<OPTIONAL_MESSAGE_CORRELATION_KEY>',
        ),
        workerJobType: read(
            'ARTIFICIALFLOW_WORKER_JOB_TYPE',
            'FLOWGO_WORKER_JOB_TYPE',
            '<OPTIONAL_SERVICE_TASK_JOB_TYPE>',
        ),
    };
}

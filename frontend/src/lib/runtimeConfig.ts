export type RuntimeConfig = {
  apiUrl?: string;
  oidcAuthority?: string;
  oidcClientId?: string;
};

declare global {
  interface Window {
    __ARTIFICIALFLOW_RUNTIME_CONFIG__?: RuntimeConfig;
  }
}

const trim = (value: string | undefined | null): string => (value || "").trim();

export function resolveRuntimeConfig(canonical: RuntimeConfig = {}) {
  return {
    apiUrl: trim(canonical.apiUrl),
    oidcAuthority: trim(canonical.oidcAuthority),
    oidcClientId: trim(canonical.oidcClientId),
  };
}

const canonicalRuntime =
  (typeof window !== "undefined" ? window.__ARTIFICIALFLOW_RUNTIME_CONFIG__ : undefined) || {};

export const runtimeConfig = resolveRuntimeConfig(canonicalRuntime);

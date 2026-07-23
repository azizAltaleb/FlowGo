export type RuntimeConfig = {
  apiUrl?: string;
  oidcAuthority?: string;
  oidcClientId?: string;
};

declare global {
  interface Window {
    __ARTIFICIALFLOW_RUNTIME_CONFIG__?: RuntimeConfig;
    __FLOWGO_RUNTIME_CONFIG__?: RuntimeConfig;
  }
}

const trim = (value: string | undefined | null): string => (value || "").trim();

export function resolveRuntimeConfig(
  canonical: RuntimeConfig = {},
  legacy: RuntimeConfig = {},
) {
  const preferred = (canonicalValue?: string, legacyValue?: string) =>
    trim(canonicalValue) || trim(legacyValue);

  return {
    apiUrl: preferred(canonical.apiUrl, legacy.apiUrl),
    oidcAuthority: preferred(canonical.oidcAuthority, legacy.oidcAuthority),
    oidcClientId: preferred(canonical.oidcClientId, legacy.oidcClientId),
  };
}

const canonicalRuntime =
  (typeof window !== "undefined" ? window.__ARTIFICIALFLOW_RUNTIME_CONFIG__ : undefined) || {};
const legacyRuntime =
  (typeof window !== "undefined" ? window.__FLOWGO_RUNTIME_CONFIG__ : undefined) || {};

export const runtimeConfig = resolveRuntimeConfig(canonicalRuntime, legacyRuntime);

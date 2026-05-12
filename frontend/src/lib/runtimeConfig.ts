type RuntimeConfig = {
  apiUrl?: string;
  oidcAuthority?: string;
  oidcClientId?: string;
};

declare global {
  interface Window {
    __FLOWGO_RUNTIME_CONFIG__?: RuntimeConfig;
  }
}

const trim = (value: string | undefined | null): string => (value || "").trim();

const runtime = (typeof window !== "undefined" ? window.__FLOWGO_RUNTIME_CONFIG__ : undefined) || {};

export const runtimeConfig = {
  apiUrl: trim(runtime.apiUrl),
  oidcAuthority: trim(runtime.oidcAuthority),
  oidcClientId: trim(runtime.oidcClientId),
};

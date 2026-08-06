import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { AuthProvider } from "react-oidc-context";
import { WebStorageStateStore } from "oidc-client-ts";
import './index.css'
import App from './App.tsx'
import { runtimeConfig } from './lib/runtimeConfig';

const authority = runtimeConfig.oidcAuthority || import.meta.env.VITE_OIDC_AUTHORITY || "";
const clientID = runtimeConfig.oidcClientId || import.meta.env.VITE_OIDC_CLIENT_ID || "";

function bootstrap() {
  const authConfigured = Boolean(authority && clientID);
  const oidcConfig = {
    authority,
    client_id: clientID,
    redirect_uri: window.location.origin,
    // Must match a registered ZITADEL redirect URI (bundled bootstrap uses origin).
    silent_redirect_uri: window.location.origin,
    post_logout_redirect_uri: window.location.origin,
    response_type: "code",
    scope: "openid profile email",
    automaticSilentRenew: true,
    includeIdTokenInSilentRenew: true,
    // check_session_iframe is flaky across browsers/ZITADEL and falsely signs users out.
    monitorSession: false,
    // Survive hard refresh better than default sessionStorage.
    userStore: new WebStorageStateStore({ store: window.localStorage }),
    onSigninCallback: () => {
      // Avoid rewriting history inside the silent-renew iframe.
      if (window.self !== window.top) {
        return;
      }
      window.history.replaceState({}, document.title, window.location.pathname + window.location.hash);
    },
  };

  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      {authConfigured ? (
        <AuthProvider {...oidcConfig}>
          <App />
        </AuthProvider>
      ) : (
        <App authDisabled />
      )}
    </StrictMode>,
  )
}

bootstrap()

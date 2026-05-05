import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { AuthProvider } from "react-oidc-context";
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
    post_logout_redirect_uri: window.location.origin,
    onSigninCallback: () => {
      window.history.replaceState({}, document.title, window.location.pathname);
    }
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

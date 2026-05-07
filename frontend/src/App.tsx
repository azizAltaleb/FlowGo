import { lazy, Suspense, useEffect, useLayoutEffect } from "react";
import { BrowserRouter, Route, Routes } from "react-router-dom";
import { useAuth } from "react-oidc-context";
import { setAccessToken } from "@/lib/api";
import DashboardLayout from "@/layouts/DashboardLayout";
import { Loader2 } from "lucide-react";

const Dashboard = lazy(() => import("@/pages/Dashboard"));
const Modeler = lazy(() => import("@/pages/Modeler"));
const Processes = lazy(() => import("@/pages/Processes"));
const Instances = lazy(() => import("@/pages/Instances"));
const InstanceDetails = lazy(() => import("@/pages/InstanceDetails"));
const IdentityManagement = lazy(() => import("@/pages/IdentityManagement"));
const GoFlowClients = lazy(() => import("@/pages/GoFlowClients"));

type AppProps = {
  authDisabled?: boolean;
};

type AppRoutesProps = {
  onLogout?: () => void;
};

function LoadingScreen() {
  return (
    <div className="flex h-screen items-center justify-center">
      <Loader2 className="h-8 w-8 animate-spin text-primary" />
    </div>
  );
}

function AppRoutes({ onLogout }: AppRoutesProps) {
  return (
    <BrowserRouter>
      <Suspense fallback={<LoadingScreen />}>
        <Routes>
          <Route path="/" element={<DashboardLayout onLogout={onLogout} />}>
            <Route index element={<Dashboard />} />
            <Route path="modeler" element={<Modeler />} />
            <Route path="processes" element={<Processes />} />
            <Route path="instances" element={<Instances />} />
            <Route path="instances/:id" element={<InstanceDetails />} />
            <Route path="identity" element={<IdentityManagement />} />
            <Route path="sdk-clients" element={<GoFlowClients />} />
          </Route>
        </Routes>
      </Suspense>
    </BrowserRouter>
  );
}

function AuthenticatedApp() {
  const auth = useAuth();
  const accessToken = auth.user?.access_token;

  useLayoutEffect(() => {
    if (auth.isAuthenticated && accessToken) {
      setAccessToken(accessToken);
    } else {
      setAccessToken(null);
    }
  }, [auth.isAuthenticated, accessToken]);

  if (auth.isLoading) {
    return <LoadingScreen />;
  }

  if (auth.error) {
    return <div>Oops... {auth.error.message}</div>;
  }

  if (!auth.isAuthenticated) {
    return (
      <div className="flex h-screen items-center justify-center bg-muted/50">
        <div className="text-center space-y-4">
          <h1 className="text-2xl font-bold">GoFlow</h1>
          <p className="text-muted-foreground">Please sign in to continue</p>
          <button
            onClick={() => void auth.signinRedirect()}
            className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
          >
            Sign in
          </button>
        </div>
      </div>
    );
  }

  if (!accessToken) {
    return <LoadingScreen />;
  }

  const logout = () => {
    setAccessToken(null);
    void auth.signoutRedirect({
      post_logout_redirect_uri: window.location.origin,
    });
  };

  return <AppRoutes onLogout={logout} />;
}

function AuthDisabledApp() {
  useEffect(() => {
    setAccessToken(null);
  }, []);

  return <AppRoutes />;
}

function App({ authDisabled = false }: AppProps) {
  if (authDisabled) {
    return <AuthDisabledApp />;
  }

  return <AuthenticatedApp />;
}

export default App;

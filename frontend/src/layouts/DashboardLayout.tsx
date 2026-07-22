import { Link, Navigate, Outlet, useLocation } from "react-router-dom";
import { LayoutDashboard, Layers, Activity, ShieldUser, Menu, X, LogOut, KeyRound, History } from "lucide-react";
import { useEffect, useState } from "react";
import { cn } from "@/lib/utils";
import { api, type IdentityConfigResponse, type IdentityResponse } from "@/lib/api";
import { Button } from "@/components/ui/button";
import {
  FLOWGO_ADMIN_ROLE,
  FLOWGO_CLIENT_ROLE,
  FLOWGO_MODELER_ROLE,
  hasFlexRole,
  isAdmin,
  isClientOnly,
  isModeler,
} from "@/lib/roles";

const sidebarItems = [
  { icon: LayoutDashboard, label: "Dashboard", href: "/" },
  { icon: Layers, label: "Processes", href: "/processes" },
  { icon: Activity, label: "Instances", href: "/instances" },
  { icon: History, label: "History", href: "/history" },
  { icon: ShieldUser, label: "Identity", href: "/identity" },
  { icon: KeyRound, label: "SDK Clients", href: "/sdk-clients" },
];

type DashboardLayoutProps = {
  onLogout?: () => void;
};

export default function DashboardLayout({ onLogout }: DashboardLayoutProps) {
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);
  const [identity, setIdentity] = useState<IdentityResponse | null>(null);
  const [identityConfig, setIdentityConfig] = useState<IdentityConfigResponse | null>(null);
  const [loadingAccess, setLoadingAccess] = useState(true);
  const location = useLocation();
  const admin = isAdmin(identity);
  const modeler = isModeler(identity);
  const flexUser = hasFlexRole(identity);
  const clientOnly = isClientOnly(identity);
  const canShowIdentity =
    identityConfig?.deployment_mode === "zitadel" &&
    admin;
  const visibleSidebarItems = sidebarItems.filter((item) => {
    if (item.href === "/") {
      return admin;
    }
    if (item.href === "/processes") {
      return admin || modeler;
    }
    if (item.href === "/instances" || item.href === "/history") {
      return admin || (flexUser && !clientOnly);
    }
    if (item.href === "/identity" || item.href === "/sdk-clients") {
      return canShowIdentity;
    }
    return false;
  });

  useEffect(() => {
    let cancelled = false;
    const loadIdentityAccess = async () => {
      try {
        const [identityResponse, configResponse] = await Promise.all([
          api.getIdentity(),
          api.getIdentityConfig(),
        ]);
        if (!cancelled) {
          setIdentity(identityResponse);
          setIdentityConfig(configResponse);
        }
      } catch {
        if (!cancelled) {
          setIdentity(null);
          setIdentityConfig(null);
        }
      } finally {
        if (!cancelled) {
          setLoadingAccess(false);
        }
      }
    };
    void loadIdentityAccess();
    return () => {
      cancelled = true;
    };
  }, []);

  if (loadingAccess) {
    return <div className="flex h-screen items-center justify-center">Loading access...</div>;
  }

  if (clientOnly) {
    return (
      <div className="flex h-screen items-center justify-center bg-muted/50 p-6">
        <div className="max-w-lg rounded-lg border bg-card p-6 text-center shadow-sm">
          <h1 className="text-2xl font-bold">Machine Client Access</h1>
          <p className="mt-3 text-sm text-muted-foreground">
            The {FLOWGO_CLIENT_ROLE} role is reserved for SDK, API, worker, and transaction inbox integrations.
            Use a human account with {FLOWGO_ADMIN_ROLE}, {FLOWGO_MODELER_ROLE}, or a business role for console access.
          </p>
          {onLogout && (
            <Button className="mt-6" variant="outline" onClick={onLogout}>
              <LogOut className="mr-2 h-4 w-4" />
              Logout
            </Button>
          )}
        </div>
      </div>
    );
  }

  const firstAllowedPath = visibleSidebarItems[0]?.href || "";
  const path = location.pathname;
  const isAllowedPath =
    admin ||
    (modeler && (path === "/processes" || path.startsWith("/modeler"))) ||
    (flexUser && !modeler && (path === "/instances" || path.startsWith("/instances/") || path === "/history"));

  if (path === "/" && firstAllowedPath && firstAllowedPath !== "/") {
    return <Navigate to={firstAllowedPath} replace />;
  }

  if (!isAllowedPath) {
    return (
      <div className="flex h-screen items-center justify-center bg-muted/50 p-6">
        <div className="max-w-lg rounded-lg border bg-card p-6 text-center shadow-sm">
          <h1 className="text-2xl font-bold">Access Not Allowed</h1>
          <p className="mt-3 text-sm text-muted-foreground">
            Your FlowGo role does not allow this page. Administrators have full access,
            modelers can use Processes and Modeler, and business users can use Instances and History.
          </p>
          {firstAllowedPath ? (
            <Button className="mt-6" asChild>
              <Link to={firstAllowedPath}>Go to allowed area</Link>
            </Button>
          ) : null}
          {onLogout && (
            <Button className="mt-6 ml-2" variant="outline" onClick={onLogout}>
              <LogOut className="mr-2 h-4 w-4" />
              Logout
            </Button>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-screen bg-background text-foreground overflow-hidden">
      {/* Sidebar */}
      <aside
        className={cn(
          "bg-card border-r transition-all duration-300 ease-in-out flex flex-col",
          isSidebarOpen ? "w-64" : "w-16"
        )}
      >
        <div className="h-16 flex items-center justify-between px-4 border-b">
          {isSidebarOpen && <span className="font-bold text-lg">Workflow SA</span>}
          <button
            onClick={() => setIsSidebarOpen(!isSidebarOpen)}
            className="p-1 hover:bg-accent rounded-md"
          >
            {isSidebarOpen ? <X size={20} /> : <Menu size={20} />}
          </button>
        </div>

        <nav className="flex-1 py-4 flex flex-col gap-1 px-2">
          {visibleSidebarItems.map((item) => {
            const Icon = item.icon;
            const isActive = location.pathname === item.href;
            return (
              <Link
                key={item.href}
                to={item.href}
                className={cn(
                  "flex items-center gap-3 px-3 py-2 rounded-md transition-colors",
                  isActive
                    ? "bg-primary text-primary-foreground"
                    : "hover:bg-accent hover:text-accent-foreground",
                  !isSidebarOpen && "justify-center"
                )}
              >
                <Icon size={20} />
                {isSidebarOpen && <span>{item.label}</span>}
              </Link>
            );
          })}
        </nav>
      </aside>

      {/* Main Content */}
      <main className="flex-1 flex flex-col overflow-hidden">
        <header className="h-16 border-b flex items-center justify-between px-6 bg-card">
          <h1 className="text-xl font-semibold capitalize">
            {location.pathname === "/"
              ? "Dashboard"
              : location.pathname.substring(1).replace("-", " ")}
          </h1>
          {onLogout && (
            <Button variant="outline" size="sm" onClick={onLogout}>
              <LogOut className="mr-2 h-4 w-4" />
              Logout
            </Button>
          )}
        </header>
        <div className="flex-1 overflow-auto p-6">
          <Outlet />
        </div>
      </main>
    </div>
  );
}

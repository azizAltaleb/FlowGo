import { Link, Outlet, useLocation } from "react-router-dom";
import { LayoutDashboard, Layers, Activity, ShieldUser, Menu, X, LogOut, KeyRound } from "lucide-react";
import { useEffect, useState } from "react";
import { cn } from "@/lib/utils";
import { api, type IdentityConfigResponse, type IdentityResponse } from "@/lib/api";
import { Button } from "@/components/ui/button";

const sidebarItems = [
  { icon: LayoutDashboard, label: "Dashboard", href: "/" },
  { icon: Layers, label: "Processes", href: "/processes" },
  { icon: Activity, label: "Instances", href: "/instances" },
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
  const location = useLocation();
  const canShowIdentity =
    identityConfig?.deployment_mode === "zitadel" &&
    (identity?.principal?.roles || []).some((role) => role.toLowerCase() === "goflow admin");
  const visibleSidebarItems = sidebarItems.filter((item) => !["/identity", "/sdk-clients"].includes(item.href) || canShowIdentity);

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
      }
    };
    void loadIdentityAccess();
    return () => {
      cancelled = true;
    };
  }, []);

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

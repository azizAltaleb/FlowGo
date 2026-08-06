import { Link } from "react-router";
import { Button } from "@/components/ui/button";
import { LogOut } from "lucide-react";

type ConsoleAccessDeniedProps = {
  title?: string;
  message: string;
  firstAllowedPath?: string;
  onLogout?: () => void;
};

export function ConsoleAccessDenied({
  title = "Access Not Allowed",
  message,
  firstAllowedPath,
  onLogout,
}: ConsoleAccessDeniedProps) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/50 p-6">
      <div className="max-w-lg rounded-lg border bg-card p-6 text-center shadow-sm">
        <h1 className="text-2xl font-bold">{title}</h1>
        <p className="mt-3 text-sm text-muted-foreground">{message}</p>
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

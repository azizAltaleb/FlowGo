import { useEffect, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { api, type IdentityConfigResponse, type IdentityResponse } from "@/lib/api";
import { RefreshCw } from "lucide-react";

function DetailField({ label, value }: { label: string; value?: string }) {
  return (
    <div>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="break-all font-mono text-sm">{value || "-"}</dd>
    </div>
  );
}

function deploymentModeLabel(mode?: string) {
  switch ((mode || "").toLowerCase()) {
    case "zitadel":
      return "Bundled ZITADEL";
    case "external":
      return "External IAM";
    case "disabled":
      return "Authentication disabled";
    default:
      return mode || "Unknown";
  }
}

function composeFileForMode(mode?: string) {
  switch ((mode || "").toLowerCase()) {
    case "zitadel":
      return "docker-compose.zitadel.yml";
    case "external":
      return "docker-compose.external-iam.yml";
    default:
      return "-";
  }
}

export default function Identity() {
  const [identity, setIdentity] = useState<IdentityResponse | null>(null);
  const [config, setConfig] = useState<IdentityConfigResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadIdentity = async () => {
    setLoading(true);
    setError(null);
    try {
      const [identityResponse, configResponse] = await Promise.all([
        api.getIdentity(),
        api.getIdentityConfig(),
      ]);
      setIdentity(identityResponse);
      setConfig(configResponse);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load identity";
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadIdentity();
  }, []);

  if (loading) {
    return <div className="p-4">Loading identity...</div>;
  }

  const principal = identity?.principal;
  const roles = principal?.roles || [];
  const scopes = principal?.scopes || [];
  const standardRoles = config?.standard_roles || [];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold tracking-tight">Identity</h2>
        <Button variant="outline" size="sm" onClick={loadIdentity}>
          <RefreshCw className="mr-2 h-4 w-4" />
          Refresh
        </Button>
      </div>

      {error && (
        <Card>
          <CardContent className="pt-6 text-sm text-destructive">{error}</CardContent>
        </Card>
      )}

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>External IAM integration</CardTitle>
            <CardDescription>
              Use an existing OIDC-compatible IAM already provided by the software engineer or customer environment.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex flex-wrap gap-2">
              <Badge variant={config?.deployment_mode === "external" ? "default" : "outline"}>
                {config?.deployment_mode === "external" ? "Current mode" : "Supported mode"}
              </Badge>
              <Badge variant="outline">docker-compose.external-iam.yml</Badge>
            </div>
            <p className="text-muted-foreground">
              Point the platform at the external issuer, client IDs, and optional introspection settings directly from Docker Compose.
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Solution-managed IAM</CardTitle>
            <CardDescription>
              Run the platform with a bundled ZITADEL-based IAM deployment owned by the solution itself.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex flex-wrap gap-2">
              <Badge variant={config?.deployment_mode === "zitadel" ? "default" : "outline"}>
                {config?.deployment_mode === "zitadel" ? "Current mode" : "Supported mode"}
              </Badge>
              <Badge variant="outline">docker-compose.zitadel.yml</Badge>
            </div>
            <p className="text-muted-foreground">
              Use this mode when you want GoFlow to provide its own IAM stack with ZITADEL.
            </p>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Current deployment</CardTitle>
          <CardDescription>
            IAM settings are now compose-driven and read-only inside the application runtime.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap gap-2">
            <Badge variant={identity?.authenticated ? "default" : "secondary"}>
              {identity?.authenticated ? "Authenticated" : "Not authenticated"}
            </Badge>
            <Badge variant={config?.auth_enabled ? "default" : "secondary"}>
              {config?.auth_enabled ? "Backend auth enabled" : "Backend auth disabled"}
            </Badge>
            <Badge variant={config?.frontend_auth_enabled ? "default" : "secondary"}>
              {config?.frontend_auth_enabled ? "Frontend OIDC enabled" : "Frontend OIDC disabled"}
            </Badge>
            <Badge variant="outline">{deploymentModeLabel(config?.deployment_mode)}</Badge>
          </div>

          <dl className="grid gap-3 md:grid-cols-2">
            <DetailField label="Provider" value={config?.provider_name} />
            <DetailField label="Configuration source" value={config?.configuration_source} />
            <DetailField label="Recommended compose file" value={composeFileForMode(config?.deployment_mode)} />
            <DetailField label="Token validation mode" value={config?.token_validation_mode} />
          </dl>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>OIDC configuration</CardTitle>
          <CardDescription>
            These values come from the active deployment and show how backend and frontend authentication are wired right now.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap gap-2">
            <Badge variant={config?.enforce_audience ? "default" : "outline"}>
              {config?.enforce_audience ? "Audience enforced" : "Audience relaxed"}
            </Badge>
            <Badge variant={config?.allow_insecure_issuer ? "outline" : "default"}>
              {config?.allow_insecure_issuer ? "Issuer mismatch allowed" : "Issuer strict"}
            </Badge>
          </div>

          <dl className="grid gap-3 md:grid-cols-2">
            <DetailField label="Backend client ID" value={config?.client_id} />
            <DetailField label="Frontend client ID" value={config?.frontend_oidc_client_id} />
            <DetailField label="Internal issuer URL" value={config?.internal_issuer_url} />
            <DetailField label="External issuer URL" value={config?.external_issuer_url} />
            <DetailField label="Frontend OIDC authority" value={config?.frontend_oidc_authority} />
            <DetailField label="Introspection URL" value={config?.introspection_url} />
            <DetailField label="Introspection client ID" value={config?.introspection_client_id} />
            <DetailField label="Introspection auth method" value={config?.introspection_auth_method} />
            <DetailField label="Subject claim path" value={config?.claim_subject_path} />
            <DetailField label="Roles claim path" value={config?.claim_roles_path} />
            <DetailField label="Scopes claim path" value={config?.claim_scopes_path} />
            <DetailField label="Tenant claim path" value={config?.claim_tenant_path} />
            <DetailField label="Email claim path" value={config?.claim_email_path} />
            <DetailField label="Name claim path" value={config?.claim_name_path} />
          </dl>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Authenticated principal</CardTitle>
          <CardDescription>
            Inspect the claims that the current deployment maps from the signed-in user token.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {principal ? (
            <dl className="grid gap-3 md:grid-cols-2">
              <DetailField label="Subject" value={principal.subject} />
              <DetailField label="Issuer" value={principal.issuer} />
              <DetailField label="Name" value={principal.name} />
              <DetailField label="Email" value={principal.email} />
              <DetailField label="Tenant" value={principal.tenant_id} />
              <DetailField label="Token mode" value={principal.token_mode} />
            </dl>
          ) : (
            <div className="text-sm text-muted-foreground">No authenticated principal is currently available.</div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Roles</CardTitle>
          <CardDescription>
            GoFlow recognizes these standard IAM roles: {standardRoles.join(", ") || "-"}.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          {roles.length > 0 ? (
            roles.map((role) => (
              <Badge key={role} variant="outline" className="font-mono">
                {role}
              </Badge>
            ))
          ) : (
            <span className="text-sm text-muted-foreground">No roles mapped</span>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Scopes</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          {scopes.length > 0 ? (
            scopes.map((scope) => (
              <Badge key={scope} variant="outline" className="font-mono">
                {scope}
              </Badge>
            ))
          ) : (
            <span className="text-sm text-muted-foreground">No scopes mapped</span>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Raw claims</CardTitle>
        </CardHeader>
        <CardContent>
          <pre className="overflow-auto rounded-md border bg-muted p-3 text-xs">
            {JSON.stringify(principal?.claims || {}, null, 2)}
          </pre>
        </CardContent>
      </Card>
    </div>
  );
}

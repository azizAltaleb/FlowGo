import { useEffect, useMemo, useState, type FormEvent } from "react";
import { Navigate } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import {
  api,
  type CreateIdentityManagementClientTokenRequest,
  type IdentityConfigResponse,
  type IdentityManagementClient,
  type IdentityManagementClientToken,
  type IdentityResponse,
} from "@/lib/api";
import { Copy, KeyRound, RefreshCw, RotateCw, ShieldCheck, Trash2, Unplug } from "lucide-react";

const environments = ["sandbox", "development", "staging", "production"];

function defaultExpiration() {
  const date = new Date();
  date.setDate(date.getDate() + 90);
  date.setMinutes(date.getMinutes() - date.getTimezoneOffset());
  return date.toISOString().slice(0, 16);
}

function emptyClientForm(): CreateIdentityManagementClientTokenRequest {
  return {
    username: "",
    name: "",
    description: "",
    environment: "sandbox",
    owner_email: "",
    purpose: "",
    token_expires_at: defaultExpiration(),
  };
}

function isAdmin(identity: IdentityResponse | null) {
  return (identity?.principal?.roles || []).some((role) => role.toLowerCase() === "goflow admin");
}

function messageFromError(err: unknown) {
  return err instanceof Error ? err.message : "GoFlow client request failed";
}

function toISODateTime(value?: string) {
  const trimmed = (value || "").trim();
  if (!trimmed) return undefined;
  const date = new Date(trimmed);
  return Number.isNaN(date.getTime()) ? trimmed : date.toISOString();
}

function formatDate(value: string) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function shortID(value: string) {
  if (!value) return "-";
  return value.length <= 12 ? value : `${value.slice(0, 6)}…${value.slice(-6)}`;
}

function tokenStatusVariant(status: string, expiresAt: string) {
  if (status.toLowerCase() === "expired") return "destructive";
  const expiry = new Date(expiresAt).getTime();
  const thirtyDays = Date.now() + 30 * 24 * 60 * 60 * 1000;
  if (!Number.isNaN(expiry) && expiry <= thirtyDays) return "warning";
  return "success";
}

export default function GoFlowClients() {
  const [identity, setIdentity] = useState<IdentityResponse | null>(null);
  const [config, setConfig] = useState<IdentityConfigResponse | null>(null);
  const [clients, setClients] = useState<IdentityManagementClient[]>([]);
  const [newClient, setNewClient] = useState<CreateIdentityManagementClientTokenRequest>(emptyClientForm());
  const [createdToken, setCreatedToken] = useState<IdentityManagementClientToken | null>(null);
  const [rotateExpirations, setRotateExpirations] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const canManage = config?.deployment_mode === "zitadel" && isAdmin(identity);

  const stats = useMemo(() => {
    const tokens = clients.flatMap((client) => client.tokens || []);
    const activeTokens = tokens.filter((token) => token.status.toLowerCase() !== "expired").length;
    const expiringSoon = tokens.filter((token) => tokenStatusVariant(token.status, token.token_expires_at) === "warning").length;
    return { clients: clients.length, activeTokens, expiringSoon };
  }, [clients]);

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const [identityResponse, configResponse] = await Promise.all([api.getIdentity(), api.getIdentityConfig()]);
      setIdentity(identityResponse);
      setConfig(configResponse);
      if (configResponse.deployment_mode === "zitadel" && isAdmin(identityResponse)) {
        setClients(await api.getIdentityManagementClients());
      } else {
        setClients([]);
      }
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setLoading(false);
    }
  };

  const mutate = async (operation: () => Promise<void>, reload = true) => {
    setSaving(true);
    setError(null);
    try {
      await operation();
      if (reload) await load();
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setSaving(false);
    }
  };

  useEffect(() => {
    void load();
  }, []);

  const submitClient = async (event: FormEvent) => {
    event.preventDefault();
    await mutate(async () => {
      const token = await api.createIdentityManagementClientToken({
        username: newClient.username?.trim() || undefined,
        name: newClient.name.trim(),
        description: newClient.description?.trim() || undefined,
        environment: newClient.environment,
        owner_email: newClient.owner_email?.trim(),
        purpose: newClient.purpose?.trim(),
        token_expires_at: toISODateTime(newClient.token_expires_at),
      });
      setCreatedToken(token);
      setNewClient(emptyClientForm());
    });
  };

  const rotateToken = async (client: IdentityManagementClient) => {
    const expiresAt = rotateExpirations[client.client_id] || defaultExpiration();
    await mutate(async () => {
      const token = await api.rotateIdentityManagementClientToken(client.client_id, { token_expires_at: toISODateTime(expiresAt) });
      setCreatedToken(token);
      setRotateExpirations((current) => ({ ...current, [client.client_id]: defaultExpiration() }));
    });
  };

  const copyToken = async (value: string) => {
    try {
      await navigator.clipboard.writeText(value);
    } catch {
      setError("Unable to copy token automatically. Select and copy it manually.");
    }
  };

  if (loading) return <div className="p-4">Loading GoFlow clients...</div>;
  if (config && !canManage) return <Navigate to="/" replace />;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">SDK Clients</h2>
          <p className="text-sm text-muted-foreground">Manage GoFlow Client machine identities and one-time SDK tokens.</p>
        </div>
        <Button variant="outline" size="sm" onClick={load} disabled={saving}>
          <RefreshCw className="mr-2 h-4 w-4" />
          Refresh
        </Button>
      </div>

      {error && <Card><CardContent className="pt-6 text-sm text-destructive">{error}</CardContent></Card>}

      <div className="grid gap-4 md:grid-cols-3">
        <Card><CardHeader><CardTitle>{stats.clients}</CardTitle><CardDescription>Registered SDK clients</CardDescription></CardHeader></Card>
        <Card><CardHeader><CardTitle>{stats.activeTokens}</CardTitle><CardDescription>Active tokens</CardDescription></CardHeader></Card>
        <Card><CardHeader><CardTitle>{stats.expiringSoon}</CardTitle><CardDescription>Tokens expiring within 30 days</CardDescription></CardHeader></Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2"><ShieldCheck className="h-5 w-5" />Integration criteria</CardTitle>
          <CardDescription>Recommended controls for community-standard SDK integrations.</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 text-sm md:grid-cols-2 lg:grid-cols-3">
          <div className="rounded-md border p-3"><div className="font-medium">Least privilege</div><div className="text-muted-foreground">Clients receive only the goflow client role.</div></div>
          <div className="rounded-md border p-3"><div className="font-medium">Ownership</div><div className="text-muted-foreground">Every client has an owner email for rotation and incident response.</div></div>
          <div className="rounded-md border p-3"><div className="font-medium">Environment scoped</div><div className="text-muted-foreground">Use separate clients for sandbox, staging, and production.</div></div>
          <div className="rounded-md border p-3"><div className="font-medium">Purpose declared</div><div className="text-muted-foreground">Document which app, worker, or automation uses the token.</div></div>
          <div className="rounded-md border p-3"><div className="font-medium">Expiring credentials</div><div className="text-muted-foreground">Prefer 90-day tokens and rotate before expiry.</div></div>
          <div className="rounded-md border p-3"><div className="font-medium">One-time secret display</div><div className="text-muted-foreground">Tokens are only shown immediately after creation or rotation.</div></div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2"><KeyRound className="h-5 w-5" />Create SDK client</CardTitle>
          <CardDescription>Create a machine identity and issue its first one-time token.</CardDescription>
        </CardHeader>
        <CardContent>
          <form className="grid gap-3 md:grid-cols-2" onSubmit={submitClient}>
            <div className="space-y-2"><Label>Client name</Label><Input required value={newClient.name} onChange={(event) => setNewClient({ ...newClient, name: event.target.value })} placeholder="Orders Worker" /></div>
            <div className="space-y-2"><Label>Service username</Label><Input value={newClient.username || ""} onChange={(event) => setNewClient({ ...newClient, username: event.target.value })} placeholder="orders-worker-sdk" /></div>
            <div className="space-y-2"><Label>Environment</Label><select className="h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm" value={newClient.environment} onChange={(event) => setNewClient({ ...newClient, environment: event.target.value })}>{environments.map((environment) => <option key={environment} value={environment}>{environment}</option>)}</select></div>
            <div className="space-y-2"><Label>Owner email</Label><Input required type="email" value={newClient.owner_email || ""} onChange={(event) => setNewClient({ ...newClient, owner_email: event.target.value })} placeholder="platform@example.com" /></div>
            <div className="space-y-2"><Label>Purpose</Label><Input required value={newClient.purpose || ""} onChange={(event) => setNewClient({ ...newClient, purpose: event.target.value })} placeholder="Process order payment jobs" /></div>
            <div className="space-y-2"><Label>Token expires at</Label><Input required type="datetime-local" value={newClient.token_expires_at || ""} onChange={(event) => setNewClient({ ...newClient, token_expires_at: event.target.value })} /></div>
            <div className="space-y-2 md:col-span-2"><Label>Description</Label><Textarea value={newClient.description || ""} onChange={(event) => setNewClient({ ...newClient, description: event.target.value })} placeholder="Used by the Node.js SDK worker deployed in production." /></div>
            <Button type="submit" disabled={saving} className="md:col-span-2"><KeyRound className="mr-2 h-4 w-4" />Create client and token</Button>
          </form>
        </CardContent>
      </Card>

      {createdToken && (
        <Card className="border-amber-300 bg-amber-50 text-amber-950">
          <CardHeader>
            <CardTitle>Copy token now</CardTitle>
            <CardDescription className="text-amber-900">This secret is shown only once. Store it in your secret manager or deployment environment.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="grid gap-2 text-sm md:grid-cols-2">
              <div><span className="font-medium">Client:</span> {createdToken.name}</div>
              <div><span className="font-medium">Environment:</span> {createdToken.environment || "-"}</div>
              <div><span className="font-medium">Token ID:</span> {createdToken.token_id}</div>
              <div><span className="font-medium">Expires:</span> {formatDate(createdToken.token_expires_at)}</div>
            </div>
            <Textarea readOnly value={createdToken.token} className="font-mono" />
            <div className="flex flex-wrap gap-2">
              <Button type="button" variant="outline" size="sm" onClick={() => copyToken(createdToken.token)}><Copy className="mr-2 h-4 w-4" />Copy token</Button>
              <Button type="button" variant="outline" size="sm" onClick={() => copyToken(`GOFLOW_TOKEN=${createdToken.token}`)}><Copy className="mr-2 h-4 w-4" />Copy env var</Button>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Registered clients</CardTitle>
          <CardDescription>Rotate tokens without recreating clients, revoke old tokens, or delete unused clients.</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader><TableRow><TableHead>Client</TableHead><TableHead>Criteria</TableHead><TableHead>Tokens</TableHead><TableHead className="text-right">Actions</TableHead></TableRow></TableHeader>
            <TableBody>
              {clients.map((client) => (
                <TableRow key={client.client_id}>
                  <TableCell>
                    <div className="font-medium">{client.name}</div>
                    <div className="text-xs text-muted-foreground">{client.username || client.client_id}</div>
                    <div className="mt-1 flex flex-wrap gap-1"><Badge variant="outline">{client.role}</Badge><Badge variant={client.state.toLowerCase().includes("active") ? "success" : "warning"}>{client.state || "unknown"}</Badge></div>
                  </TableCell>
                  <TableCell>
                    <div className="space-y-1 text-sm">
                      <div><span className="font-medium">Environment:</span> {client.environment || "-"}</div>
                      <div><span className="font-medium">Owner:</span> {client.owner_email || "-"}</div>
                      <div><span className="font-medium">Purpose:</span> {client.purpose || "-"}</div>
                      {client.description && <div className="text-xs text-muted-foreground">{client.description}</div>}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="space-y-2">
                      {client.tokens.length === 0 && <span className="text-xs text-muted-foreground">No tokens</span>}
                      {client.tokens.map((token) => (
                        <div key={token.token_id} className="rounded-md border p-2 text-xs">
                          <div className="flex items-center justify-between gap-2"><span className="font-mono">{shortID(token.token_id)}</span><Badge variant={tokenStatusVariant(token.status, token.token_expires_at)}>{token.status}</Badge></div>
                          <div className="text-muted-foreground">Expires {formatDate(token.token_expires_at)}</div>
                          <Button type="button" variant="outline" size="sm" className="mt-2" disabled={saving} onClick={() => window.confirm(`Revoke token ${token.token_id}?`) && mutate(() => api.revokeIdentityManagementClientToken(client.client_id, token.token_id))}><Unplug className="mr-2 h-3 w-3" />Revoke</Button>
                        </div>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-col items-end gap-2">
                      <Input type="datetime-local" className="w-56" value={rotateExpirations[client.client_id] || defaultExpiration()} onChange={(event) => setRotateExpirations((current) => ({ ...current, [client.client_id]: event.target.value }))} />
                      <Button type="button" variant="outline" size="sm" disabled={saving} onClick={() => rotateToken(client)}><RotateCw className="mr-2 h-4 w-4" />Rotate token</Button>
                      <Button type="button" variant="destructive" size="sm" disabled={saving} onClick={() => window.confirm(`Delete client ${client.name}? This revokes all its tokens.`) && mutate(() => api.deleteIdentityManagementClient(client.client_id))}><Trash2 className="mr-2 h-4 w-4" />Delete client</Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
              {clients.length === 0 && <TableRow><TableCell colSpan={4} className="text-center text-sm text-muted-foreground">No SDK clients found</TableCell></TableRow>}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}

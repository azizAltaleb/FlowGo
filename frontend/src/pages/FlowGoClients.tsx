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
  type CreateIdentityManagementClientKeyRequest,
  type IdentityConfigResponse,
  type IdentityManagementClient,
  type IdentityManagementClientCredential,
  type IdentityResponse,
} from "@/lib/api";
import {
  createServiceAccountProfile,
  generateServiceAccountKeyPair,
  serializeServiceAccountProfile,
  type ServiceAccountProfile,
} from "@/lib/serviceAccount";
import { Copy, Download, KeyRound, RefreshCw, RotateCw, ShieldCheck, Trash2, Unplug, X } from "lucide-react";

const environments = ["sandbox", "development", "staging", "production"];

function defaultKeyExpiration() {
  const date = new Date();
  date.setDate(date.getDate() + 90);
  date.setMinutes(date.getMinutes() - date.getTimezoneOffset());
  return date.toISOString().slice(0, 16);
}

type ClientForm = Omit<CreateIdentityManagementClientKeyRequest, "public_key">;

function emptyClientForm(): ClientForm {
  return {
    username: "",
    name: "",
    description: "",
    environment: "sandbox",
    owner_email: "",
    purpose: "",
    key_expires_at: defaultKeyExpiration(),
  };
}

function isAdmin(identity: IdentityResponse | null) {
  return (identity?.principal?.roles || []).some((role) => role.toLowerCase() === "flowgo admin");
}

function messageFromError(err: unknown) {
  return err instanceof Error ? err.message : "FlowGo client request failed";
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

function credentialStatusVariant(status: string, expiresAt: string) {
  if (status.toLowerCase() === "expired") return "destructive";
  const expiry = new Date(expiresAt).getTime();
  const thirtyDays = Date.now() + 30 * 24 * 60 * 60 * 1000;
  if (!Number.isNaN(expiry) && expiry <= thirtyDays) return "warning";
  return "success";
}

function profileFromKey(
  clientId: string,
  key: IdentityManagementClientCredential,
  config: IdentityConfigResponse,
  privateKey: string,
) {
  const issuer = config.frontend_oidc_authority.replace(/\/+$/, "");
  return createServiceAccountProfile(
    {
      keyId: key.id,
      userId: clientId,
      issuer,
      tokenUrl: `${issuer}/oauth/v2/token`,
      scopes: [
        "openid",
        "urn:zitadel:iam:org:projects:roles",
        `urn:zitadel:iam:org:project:id:${config.client_id}:aud`,
      ],
    },
    privateKey,
  );
}

function profileFilename(name: string) {
  const safeName = name.trim().toLowerCase().replace(/[^a-z0-9._-]+/g, "-").replace(/^-+|-+$/g, "");
  return safeName || "flowgo-client";
}

export default function FlowGoClients() {
  const [identity, setIdentity] = useState<IdentityResponse | null>(null);
  const [config, setConfig] = useState<IdentityConfigResponse | null>(null);
  const [clients, setClients] = useState<IdentityManagementClient[]>([]);
  const [newClient, setNewClient] = useState<ClientForm>(emptyClientForm());
  const [createdProfile, setCreatedProfile] = useState<{ profile: ServiceAccountProfile; name: string } | null>(null);
  const [rotateExpirations, setRotateExpirations] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const canManage = config?.deployment_mode === "zitadel" && isAdmin(identity);

  const stats = useMemo(() => {
    const credentials = clients.flatMap((client) => client.credentials || []);
    const keys = credentials.filter((credential) => credential.type === "private_key_jwt");
    const legacyPATs = credentials.filter((credential) => credential.type === "legacy_pat");
    const activeKeys = keys.filter((key) => key.status.toLowerCase() !== "expired").length;
    return { clients: clients.length, activeKeys, legacyPATs: legacyPATs.length };
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
      const generated = await generateServiceAccountKeyPair();
      const key = await api.createIdentityManagementClientKey({
        username: newClient.username?.trim() || undefined,
        name: newClient.name.trim(),
        description: newClient.description?.trim() || undefined,
        environment: newClient.environment,
        owner_email: newClient.owner_email?.trim(),
        purpose: newClient.purpose?.trim(),
        public_key: generated.publicKeyPem,
        key_expires_at: toISODateTime(newClient.key_expires_at),
      });
      const credential = key.credentials.find((item) => item.type === "private_key_jwt");
      if (!credential || !config) throw new Error("FlowGo did not return the registered key metadata.");
      setCreatedProfile({
        profile: profileFromKey(key.client_id, credential, config, generated.privateKeyPem),
        name: key.name,
      });
      setNewClient(emptyClientForm());
    });
  };

  const addKey = async (client: IdentityManagementClient) => {
    const expiresAt = rotateExpirations[client.client_id] || defaultKeyExpiration();
    await mutate(async () => {
      const generated = await generateServiceAccountKeyPair();
      const key = await api.addIdentityManagementClientKey(client.client_id, {
        public_key: generated.publicKeyPem,
        key_expires_at: toISODateTime(expiresAt),
      });
      if (!config) throw new Error("Identity configuration is unavailable.");
      setCreatedProfile({
        profile: profileFromKey(client.client_id, key, config, generated.privateKeyPem),
        name: client.name,
      });
      setRotateExpirations((current) => ({ ...current, [client.client_id]: defaultKeyExpiration() }));
    });
  };

  const copyProfile = async () => {
    if (!createdProfile) return;
    try {
      await navigator.clipboard.writeText(serializeServiceAccountProfile(createdProfile.profile));
    } catch {
      setError("Unable to copy the profile automatically. Select and copy it manually.");
    }
  };

  const downloadProfile = () => {
    if (!createdProfile) return;
    const blob = new Blob([serializeServiceAccountProfile(createdProfile.profile)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = `${profileFilename(createdProfile.name)}.service-account.json`;
    anchor.click();
    URL.revokeObjectURL(url);
    setCreatedProfile(null);
  };

  if (loading) return <div className="p-4">Loading FlowGo clients...</div>;
  if (config && !canManage) return <Navigate to="/" replace />;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">SDK Clients</h2>
          <p className="text-sm text-muted-foreground">Manage short-lived SDK authentication with browser-generated service-account keys.</p>
        </div>
        <Button variant="outline" size="sm" onClick={load} disabled={saving}>
          <RefreshCw className="mr-2 h-4 w-4" />
          Refresh
        </Button>
      </div>

      {error && <Card><CardContent className="pt-6 text-sm text-destructive">{error}</CardContent></Card>}

      <div className="grid gap-4 md:grid-cols-3">
        <Card><CardHeader><CardTitle>{stats.clients}</CardTitle><CardDescription>Registered SDK clients</CardDescription></CardHeader></Card>
        <Card><CardHeader><CardTitle>{stats.activeKeys}</CardTitle><CardDescription>Active public keys</CardDescription></CardHeader></Card>
        <Card><CardHeader><CardTitle>{stats.legacyPATs}</CardTitle><CardDescription>Legacy PATs awaiting migration</CardDescription></CardHeader></Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2"><ShieldCheck className="h-5 w-5" />Integration criteria</CardTitle>
          <CardDescription>Recommended controls for community-standard SDK integrations.</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 text-sm md:grid-cols-2 lg:grid-cols-3">
          <div className="rounded-md border p-3"><div className="font-medium">Least privilege</div><div className="text-muted-foreground">Clients receive only the flowgo client role.</div></div>
          <div className="rounded-md border p-3"><div className="font-medium">Ownership</div><div className="text-muted-foreground">Every client has an owner email for rotation and incident response.</div></div>
          <div className="rounded-md border p-3"><div className="font-medium">Environment scoped</div><div className="text-muted-foreground">Use separate clients for sandbox, staging, and production.</div></div>
          <div className="rounded-md border p-3"><div className="font-medium">Purpose declared</div><div className="text-muted-foreground">Document which app, worker, or automation uses the token.</div></div>
          <div className="rounded-md border p-3"><div className="font-medium">Overlapping rotation</div><div className="text-muted-foreground">Add and deploy a replacement key before revoking the old key.</div></div>
          <div className="rounded-md border p-3"><div className="font-medium">Browser-only private key</div><div className="text-muted-foreground">FlowGo receives only the public key. The private profile is available once.</div></div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2"><KeyRound className="h-5 w-5" />Create SDK client</CardTitle>
          <CardDescription>Generate an RSA-2048 key in this browser and register only its public half.</CardDescription>
        </CardHeader>
        <CardContent>
          <form className="grid gap-3 md:grid-cols-2" onSubmit={submitClient}>
            <div className="space-y-2"><Label>Client name</Label><Input required value={newClient.name} onChange={(event) => setNewClient({ ...newClient, name: event.target.value })} placeholder="Orders Worker" /></div>
            <div className="space-y-2"><Label>Service username</Label><Input value={newClient.username || ""} onChange={(event) => setNewClient({ ...newClient, username: event.target.value })} placeholder="orders-worker-sdk" /></div>
            <div className="space-y-2"><Label>Environment</Label><select className="h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm" value={newClient.environment} onChange={(event) => setNewClient({ ...newClient, environment: event.target.value })}>{environments.map((environment) => <option key={environment} value={environment}>{environment}</option>)}</select></div>
            <div className="space-y-2"><Label>Owner email</Label><Input required type="email" value={newClient.owner_email || ""} onChange={(event) => setNewClient({ ...newClient, owner_email: event.target.value })} placeholder="platform@example.com" /></div>
            <div className="space-y-2"><Label>Purpose</Label><Input required value={newClient.purpose || ""} onChange={(event) => setNewClient({ ...newClient, purpose: event.target.value })} placeholder="Process order payment jobs" /></div>
            <div className="space-y-2"><Label>Key expires at</Label><Input required type="datetime-local" value={newClient.key_expires_at || ""} onChange={(event) => setNewClient({ ...newClient, key_expires_at: event.target.value })} /></div>
            <div className="space-y-2 md:col-span-2"><Label>Description</Label><Textarea value={newClient.description || ""} onChange={(event) => setNewClient({ ...newClient, description: event.target.value })} placeholder="Used by the Node.js SDK worker deployed in production." /></div>
            <Button type="submit" disabled={saving} className="md:col-span-2"><KeyRound className="mr-2 h-4 w-4" />Create client and service-account profile</Button>
          </form>
        </CardContent>
      </Card>

      {createdProfile && (
        <Card className="border-amber-300 bg-amber-50 text-amber-950">
          <CardHeader>
            <CardTitle>Save this service-account profile now</CardTitle>
            <CardDescription className="text-amber-900">The private key exists only in this page and cannot be recovered. Store the JSON in a secret manager. Download or dismiss clears it from memory.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="grid gap-2 text-sm md:grid-cols-2">
              <div><span className="font-medium">Client:</span> {createdProfile.name}</div>
              <div><span className="font-medium">Key ID:</span> {createdProfile.profile.keyId}</div>
              <div><span className="font-medium">User ID:</span> {createdProfile.profile.userId}</div>
              <div><span className="font-medium">Token endpoint:</span> {createdProfile.profile.tokenUrl}</div>
            </div>
            <Textarea readOnly value={serializeServiceAccountProfile(createdProfile.profile)} className="min-h-52 font-mono text-xs" />
            <div className="flex flex-wrap gap-2">
              <Button type="button" variant="outline" size="sm" onClick={downloadProfile}><Download className="mr-2 h-4 w-4" />Download JSON</Button>
              <Button type="button" variant="outline" size="sm" onClick={copyProfile}><Copy className="mr-2 h-4 w-4" />Copy JSON</Button>
              <Button type="button" variant="outline" size="sm" onClick={() => setCreatedProfile(null)}><X className="mr-2 h-4 w-4" />Dismiss and clear</Button>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Registered clients</CardTitle>
          <CardDescription>Add a replacement key, deploy its downloaded profile, then revoke the old key. Legacy PATs can only be inspected and revoked.</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader><TableRow><TableHead>Client</TableHead><TableHead>Criteria</TableHead><TableHead>Public keys</TableHead><TableHead>Legacy PATs</TableHead><TableHead className="text-right">Actions</TableHead></TableRow></TableHeader>
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
                      {(client.credentials || []).filter((credential) => credential.type === "private_key_jwt").length === 0 && <span className="text-xs text-muted-foreground">No keys</span>}
                      {(client.credentials || []).filter((credential) => credential.type === "private_key_jwt").map((key) => (
                        <div key={key.id} className="rounded-md border p-2 text-xs">
                          <div className="flex items-center justify-between gap-2"><span className="font-mono">{shortID(key.id)}</span><Badge variant={credentialStatusVariant(key.status, key.expires_at)}>{key.status}</Badge></div>
                          <div className="text-muted-foreground">Expires {formatDate(key.expires_at)}</div>
                          <Button type="button" variant="outline" size="sm" className="mt-2" disabled={saving} onClick={() => window.confirm(`Revoke key ${key.id}? New token exchanges using it will fail.`) && mutate(() => api.revokeIdentityManagementClientKey(client.client_id, key.id))}><Unplug className="mr-2 h-3 w-3" />Revoke key</Button>
                        </div>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="space-y-2">
                      {(client.credentials || []).filter((credential) => credential.type === "legacy_pat").length === 0 && <span className="text-xs text-muted-foreground">None</span>}
                      {(client.credentials || []).filter((credential) => credential.type === "legacy_pat").map((token) => (
                        <div key={token.id} className="rounded-md border p-2 text-xs">
                          <div className="flex items-center justify-between gap-2"><span className="font-mono">{shortID(token.id)}</span><Badge variant={credentialStatusVariant(token.status, token.expires_at)}>{token.status}</Badge></div>
                          <div className="text-muted-foreground">Expires {formatDate(token.expires_at)}</div>
                          <Button type="button" variant="outline" size="sm" className="mt-2" disabled={saving} onClick={() => window.confirm(`Revoke legacy PAT ${token.id}?`) && mutate(() => api.revokeIdentityManagementClientToken(client.client_id, token.id))}><Unplug className="mr-2 h-3 w-3" />Revoke PAT</Button>
                        </div>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-col items-end gap-2">
                      <Input type="datetime-local" className="w-56" value={rotateExpirations[client.client_id] || defaultKeyExpiration()} onChange={(event) => setRotateExpirations((current) => ({ ...current, [client.client_id]: event.target.value }))} />
                      <Button type="button" variant="outline" size="sm" disabled={saving} onClick={() => addKey(client)}><RotateCw className="mr-2 h-4 w-4" />Add replacement key</Button>
                      <Button type="button" variant="destructive" size="sm" disabled={saving} onClick={() => window.confirm(`Delete client ${client.name}? This revokes all keys and legacy PATs.`) && mutate(() => api.deleteIdentityManagementClient(client.client_id))}><Trash2 className="mr-2 h-4 w-4" />Delete client</Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
              {clients.length === 0 && <TableRow><TableCell colSpan={5} className="text-center text-sm text-muted-foreground">No SDK clients found</TableCell></TableRow>}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}

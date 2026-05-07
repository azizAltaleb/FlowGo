import { useEffect, useState, type FormEvent } from "react";
import { Navigate } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import {
  api,
  type CreateIdentityManagementUserRequest,
  type IdentityConfigResponse,
  type IdentityManagementRole,
  type IdentityManagementUser,
  type IdentityResponse,
  type UpdateIdentityManagementRoleRequest,
  type UpdateIdentityManagementUserRequest,
} from "@/lib/api";
import { Pencil, Plus, RefreshCw, RotateCcw, Save, Trash2, UserX } from "lucide-react";

const emptyUser: CreateIdentityManagementUserRequest = {
  username: "",
  given_name: "",
  family_name: "",
  email: "",
  password: "",
  password_change_required: true,
  roles: [],
};

function isAdmin(identity: IdentityResponse | null) {
  return (identity?.principal?.roles || []).some((role) => role.toLowerCase() === "goflow admin");
}

function roleToggle(values: string[], role: string) {
  return values.includes(role) ? values.filter((value) => value !== role) : [...values, role];
}

function stateVariant(state: string) {
  const value = state.toLowerCase();
  if (value.includes("active")) return "success";
  if (value.includes("deactivated") || value.includes("locked")) return "warning";
  if (value.includes("deleted")) return "destructive";
  return "outline";
}

function messageFromError(err: unknown) {
  return err instanceof Error ? err.message : "Identity management request failed";
}

function RolePicker({ roles, selected, onToggle }: { roles: string[]; selected: string[]; onToggle: (role: string) => void }) {
  return (
    <div className="space-y-2">
      <Label>Roles</Label>
      <div className="flex flex-wrap gap-3">
        {roles.map((role) => (
          <label key={role} className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={selected.includes(role)} onChange={() => onToggle(role)} />
            {role}
          </label>
        ))}
        {roles.length === 0 && <span className="text-sm text-muted-foreground">No roles available</span>}
      </div>
    </div>
  );
}

export default function IdentityManagement() {
  const [identity, setIdentity] = useState<IdentityResponse | null>(null);
  const [config, setConfig] = useState<IdentityConfigResponse | null>(null);
  const [users, setUsers] = useState<IdentityManagementUser[]>([]);
  const [roles, setRoles] = useState<IdentityManagementRole[]>([]);
  const [newUser, setNewUser] = useState<CreateIdentityManagementUserRequest>(emptyUser);
  const [editingUserId, setEditingUserId] = useState<string | null>(null);
  const [editingUser, setEditingUser] = useState<UpdateIdentityManagementUserRequest>({});
  const [editingRoleKey, setEditingRoleKey] = useState<string | null>(null);
  const [editingRole, setEditingRole] = useState<UpdateIdentityManagementRoleRequest>({ display_name: "", group: "" });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const canManage = config?.deployment_mode === "zitadel" && isAdmin(identity);
  const roleKeys = roles.map((role) => role.key);

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const [identityResponse, configResponse] = await Promise.all([api.getIdentity(), api.getIdentityConfig()]);
      setIdentity(identityResponse);
      setConfig(configResponse);
      if (configResponse.deployment_mode === "zitadel" && isAdmin(identityResponse)) {
        const [usersResponse, rolesResponse] = await Promise.all([
          api.getIdentityManagementUsers(),
          api.getIdentityManagementRoles(),
        ]);
        setUsers(usersResponse);
        setRoles(rolesResponse);
      } else {
        setUsers([]);
        setRoles([]);
      }
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setLoading(false);
    }
  };

  const mutate = async (operation: () => Promise<void>) => {
    setSaving(true);
    setError(null);
    try {
      await operation();
      await load();
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setSaving(false);
    }
  };

  useEffect(() => {
    void load();
  }, []);

  const submitUser = async (event: FormEvent) => {
    event.preventDefault();
    await mutate(async () => {
      await api.createIdentityManagementUser(newUser);
      setNewUser(emptyUser);
    });
  };

  const submitUserEdit = async (event: FormEvent) => {
    event.preventDefault();
    if (!editingUserId) return;
    await mutate(async () => {
      await api.updateIdentityManagementUser(editingUserId, editingUser);
      setEditingUserId(null);
      setEditingUser({});
    });
  };

  const submitRoleEdit = async (event: FormEvent) => {
    event.preventDefault();
    if (!editingRoleKey) return;
    await mutate(async () => {
      await api.updateIdentityManagementRole(editingRoleKey, editingRole);
      setEditingRoleKey(null);
    });
  };

  if (loading) return <div className="p-4">Loading identity...</div>;
  if (config && !canManage) return <Navigate to="/" replace />;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Identity</h2>
          <p className="text-sm text-muted-foreground">Manage bundled ZITADEL users and GoFlow role assignments.</p>
        </div>
        <Button variant="outline" size="sm" onClick={load} disabled={saving}>
          <RefreshCw className="mr-2 h-4 w-4" />
          Refresh
        </Button>
      </div>

      {error && <Card><CardContent className="pt-6 text-sm text-destructive">{error}</CardContent></Card>}

      <Card>
        <CardHeader>
          <CardTitle>Access</CardTitle>
          <CardDescription>This screen is available only in bundled ZITADEL mode for goflow admin users.</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          <Badge>Bundled ZITADEL</Badge>
          <Badge variant="success">goflow admin</Badge>
          <Badge variant="outline">{identity?.principal?.email || identity?.principal?.subject}</Badge>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Add user</CardTitle>
          <CardDescription>Create a human ZITADEL user and assign GoFlow roles.</CardDescription>
        </CardHeader>
        <CardContent>
          <form className="grid gap-3 md:grid-cols-2" onSubmit={submitUser}>
            <Input placeholder="Username (optional)" value={newUser.username} onChange={(event) => setNewUser({ ...newUser, username: event.target.value })} />
            <Input required type="email" placeholder="Email" value={newUser.email} onChange={(event) => setNewUser({ ...newUser, email: event.target.value })} />
            <Input required placeholder="Given name" value={newUser.given_name} onChange={(event) => setNewUser({ ...newUser, given_name: event.target.value })} />
            <Input required placeholder="Family name" value={newUser.family_name} onChange={(event) => setNewUser({ ...newUser, family_name: event.target.value })} />
            <Input required type="password" placeholder="Initial password" value={newUser.password} onChange={(event) => setNewUser({ ...newUser, password: event.target.value })} />
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={newUser.password_change_required} onChange={(event) => setNewUser({ ...newUser, password_change_required: event.target.checked })} />
              Require password change
            </label>
            <div className="md:col-span-2">
              <RolePicker roles={roleKeys} selected={newUser.roles} onToggle={(role) => setNewUser({ ...newUser, roles: roleToggle(newUser.roles, role) })} />
            </div>
            <Button type="submit" disabled={saving} className="md:col-span-2"><Plus className="mr-2 h-4 w-4" />Add user</Button>
          </form>
        </CardContent>
      </Card>

      {editingUserId && (
        <Card>
          <CardHeader><CardTitle>Edit user</CardTitle><CardDescription>Update profile fields and role assignment.</CardDescription></CardHeader>
          <CardContent>
            <form className="grid gap-3 md:grid-cols-2" onSubmit={submitUserEdit}>
              <Input placeholder="Username" value={editingUser.username || ""} onChange={(event) => setEditingUser({ ...editingUser, username: event.target.value })} />
              <Input type="email" placeholder="Email" value={editingUser.email || ""} onChange={(event) => setEditingUser({ ...editingUser, email: event.target.value })} />
              <Input placeholder="Given name" value={editingUser.given_name || ""} onChange={(event) => setEditingUser({ ...editingUser, given_name: event.target.value })} />
              <Input placeholder="Family name" value={editingUser.family_name || ""} onChange={(event) => setEditingUser({ ...editingUser, family_name: event.target.value })} />
              <Input placeholder="Display name" value={editingUser.display_name || ""} onChange={(event) => setEditingUser({ ...editingUser, display_name: event.target.value })} className="md:col-span-2" />
              <div className="md:col-span-2"><RolePicker roles={roleKeys} selected={editingUser.roles || []} onToggle={(role) => setEditingUser({ ...editingUser, roles: roleToggle(editingUser.roles || [], role) })} /></div>
              <div className="flex gap-2 md:col-span-2">
                <Button type="submit" disabled={saving}><Save className="mr-2 h-4 w-4" />Save user</Button>
                <Button type="button" variant="outline" onClick={() => setEditingUserId(null)}>Cancel</Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader><CardTitle>Users</CardTitle><CardDescription>View, update, terminate, reactivate, or delete users.</CardDescription></CardHeader>
        <CardContent>
          <Table>
            <TableHeader><TableRow><TableHead>User</TableHead><TableHead>State</TableHead><TableHead>Roles</TableHead><TableHead className="text-right">Actions</TableHead></TableRow></TableHeader>
            <TableBody>
              {users.map((user) => (
                <TableRow key={user.id}>
                  <TableCell><div className="font-medium">{user.display_name || user.preferred_login_name}</div><div className="text-xs text-muted-foreground">{user.email || user.preferred_login_name}</div><div className="text-xs text-muted-foreground">{user.type}</div></TableCell>
                  <TableCell><Badge variant={stateVariant(user.state)}>{user.state || "-"}</Badge></TableCell>
                  <TableCell><div className="flex flex-wrap gap-1">{user.roles.length ? user.roles.map((role) => <Badge key={role} variant="outline">{role}</Badge>) : <span className="text-xs text-muted-foreground">No roles</span>}</div></TableCell>
                  <TableCell><div className="flex justify-end gap-2">
                    <Button variant="outline" size="sm" onClick={() => { setEditingUserId(user.id); setEditingUser({ username: user.username, given_name: user.given_name, family_name: user.family_name, display_name: user.display_name, email: user.email, roles: user.roles }); }}><Pencil className="h-4 w-4" /></Button>
                    {user.state === "USER_STATE_DEACTIVATED" ? <Button variant="outline" size="sm" disabled={saving} onClick={() => mutate(() => api.reactivateIdentityManagementUser(user.id))}><RotateCcw className="h-4 w-4" /></Button> : <Button variant="outline" size="sm" disabled={saving} onClick={() => mutate(() => api.terminateIdentityManagementUser(user.id))}><UserX className="h-4 w-4" /></Button>}
                    <Button variant="destructive" size="sm" disabled={saving} onClick={() => window.confirm(`Delete user ${user.preferred_login_name}?`) && mutate(() => api.deleteIdentityManagementUser(user.id))}><Trash2 className="h-4 w-4" /></Button>
                  </div></TableCell>
                </TableRow>
              ))}
              {users.length === 0 && <TableRow><TableCell colSpan={4} className="text-center text-sm text-muted-foreground">No users found</TableCell></TableRow>}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>Roles</CardTitle><CardDescription>View, update, and delete GoFlow project roles.</CardDescription></CardHeader>
        <CardContent className="space-y-4">
          {editingRoleKey && <form className="grid gap-3 md:grid-cols-4" onSubmit={submitRoleEdit}><Input value={editingRoleKey} disabled /><Input required value={editingRole.display_name} onChange={(event) => setEditingRole({ ...editingRole, display_name: event.target.value })} /><Input value={editingRole.group} onChange={(event) => setEditingRole({ ...editingRole, group: event.target.value })} /><div className="flex gap-2"><Button type="submit" disabled={saving}>Save</Button><Button type="button" variant="outline" onClick={() => setEditingRoleKey(null)}>Cancel</Button></div></form>}
          <Table>
            <TableHeader><TableRow><TableHead>Key</TableHead><TableHead>Display name</TableHead><TableHead>Group</TableHead><TableHead className="text-right">Actions</TableHead></TableRow></TableHeader>
            <TableBody>
              {roles.map((role) => <TableRow key={role.key}><TableCell className="font-mono">{role.key}</TableCell><TableCell>{role.display_name}</TableCell><TableCell>{role.group || "-"}</TableCell><TableCell><div className="flex justify-end gap-2"><Button variant="outline" size="sm" onClick={() => { setEditingRoleKey(role.key); setEditingRole({ display_name: role.display_name, group: role.group }); }}><Pencil className="h-4 w-4" /></Button><Button variant="destructive" size="sm" disabled={saving} onClick={() => window.confirm(`Delete role ${role.key}?`) && mutate(() => api.deleteIdentityManagementRole(role.key))}><Trash2 className="h-4 w-4" /></Button></div></TableCell></TableRow>)}
              {roles.length === 0 && <TableRow><TableCell colSpan={4} className="text-center text-sm text-muted-foreground">No roles found</TableCell></TableRow>}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}

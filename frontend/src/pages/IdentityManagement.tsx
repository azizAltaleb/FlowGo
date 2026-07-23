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
  type CreateIdentityManagementRoleRequest,
  type CreateIdentityManagementUserRequest,
  type IdentityConfigResponse,
  type IdentityManagementRole,
  type IdentityManagementUser,
  type IdentityResponse,
  type UpdateIdentityManagementRoleRequest,
  type UpdateIdentityManagementUserRequest,
} from "@/lib/api";
import {
  ARTIFICIALFLOW_ADMIN_ROLE,
  ARTIFICIALFLOW_CLIENT_ROLE,
  ARTIFICIALFLOW_MODELER_ROLE,
  STATIC_ARTIFICIALFLOW_ROLES,
  canonicalizeRole,
  isAdmin,
} from "@/lib/roles";
import { Pencil, Plus, RefreshCw, Save, Trash2, UserCheck, UserX } from "lucide-react";

const emptyUser: CreateIdentityManagementUserRequest = {
  username: "",
  given_name: "",
  family_name: "",
  email: "",
  password: "",
  password_change_required: true,
  roles: [],
};

const emptyRole: CreateIdentityManagementRoleRequest = {
  key: "",
  display_name: "",
  group: "Business",
};

const FLOWGO_VIEWER_ROLE = "flowgo viewer";
const ARTIFICIALFLOW_VIEWER_ROLE = "artificialflow viewer";
const HUMAN_PLATFORM_ROLE_KEYS = [ARTIFICIALFLOW_ADMIN_ROLE, ARTIFICIALFLOW_MODELER_ROLE];

function isPlatformRole(role: string) {
  return STATIC_ARTIFICIALFLOW_ROLES.includes(canonicalizeRole(role));
}

function isReservedRole(role: string) {
  const normalized = role.trim().toLowerCase();
  return isPlatformRole(role) || normalized === FLOWGO_VIEWER_ROLE || normalized === ARTIFICIALFLOW_VIEWER_ROLE;
}

function roleToggle(values: string[], role: string) {
  return values.includes(role) ? values.filter((value) => value !== role) : [...values, role];
}

function stateVariant(state: string) {
  const value = state.toLowerCase();
  if (isInactiveUserState(state)) return "warning";
  if (value === "active" || value.endsWith("_active")) return "success";
  if (value.includes("deleted")) return "destructive";
  return "outline";
}

function isInactiveUserState(state: string) {
  const value = state.toLowerCase();
  return (
    value.includes("inactive") ||
    value.includes("deactivated") ||
    value.includes("locked") ||
    value.includes("suspended")
  );
}

function messageFromError(err: unknown) {
  return err instanceof Error ? err.message : "Identity management request failed";
}

function RolePicker({
  platformRoles,
  customRoles,
  selected,
  onToggle,
}: {
  platformRoles: string[];
  customRoles: string[];
  selected: string[];
  onToggle: (role: string) => void;
}) {
  return (
    <div className="space-y-2">
      <Label>Roles</Label>
      <div className="space-y-1">
        <div className="text-xs font-medium text-muted-foreground">Platform roles</div>
        <div className="flex flex-wrap gap-3">
          {platformRoles.map((role) => (
            <label key={role} className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={selected.includes(role)} onChange={() => onToggle(role)} />
              {role}
            </label>
          ))}
        </div>
      </div>
      <div className="text-xs font-medium text-muted-foreground">Business roles</div>
      <div className="flex flex-wrap gap-3">
        {customRoles.map((role) => (
          <label key={role} className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={selected.includes(role)} onChange={() => onToggle(role)} />
            {role}
          </label>
        ))}
        {customRoles.length === 0 && (
          <span className="text-sm text-muted-foreground">No custom business roles available</span>
        )}
      </div>
      {selected.includes(ARTIFICIALFLOW_ADMIN_ROLE) && (
        <p className="text-xs text-muted-foreground">
          {ARTIFICIALFLOW_ADMIN_ROLE} is the administrative static role and remains assigned to current admins.
        </p>
      )}
      {selected.includes(ARTIFICIALFLOW_MODELER_ROLE) && (
        <p className="text-xs text-muted-foreground">
          {ARTIFICIALFLOW_MODELER_ROLE} can read and deploy process definitions, but cannot start instances or manage identity.
        </p>
      )}
    </div>
  );
}

export default function IdentityManagement() {
  const [identity, setIdentity] = useState<IdentityResponse | null>(null);
  const [config, setConfig] = useState<IdentityConfigResponse | null>(null);
  const [users, setUsers] = useState<IdentityManagementUser[]>([]);
  const [roles, setRoles] = useState<IdentityManagementRole[]>([]);
  const [newUser, setNewUser] = useState<CreateIdentityManagementUserRequest>(emptyUser);
  const [newRole, setNewRole] = useState<CreateIdentityManagementRoleRequest>(emptyRole);
  const [editingUserId, setEditingUserId] = useState<string | null>(null);
  const [editingUser, setEditingUser] = useState<UpdateIdentityManagementUserRequest>({});
  const [editingRoleKey, setEditingRoleKey] = useState<string | null>(null);
  const [editingRole, setEditingRole] = useState<UpdateIdentityManagementRoleRequest>({ display_name: "", group: "" });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const canManage = config?.deployment_mode === "zitadel" && isAdmin(identity);
  const roleKeys = roles.map((role) => role.key);
  const platformRoleKeys = HUMAN_PLATFORM_ROLE_KEYS.filter((role) =>
    roleKeys.some((roleKey) => canonicalizeRole(roleKey) === role),
  );
  const customRoleKeys = roleKeys.filter((role) => !isReservedRole(role));

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

  const submitRole = async (event: FormEvent) => {
    event.preventDefault();
    if (isReservedRole(newRole.key)) {
      setError("ArtificialFlow admin, artificialflow modeler, and artificialflow client are built-in platform roles. Legacy flowgo roles are reserved during migration.");
      return;
    }
    await mutate(async () => {
      await api.createIdentityManagementRole(newRole);
      setNewRole(emptyRole);
    });
  };

  if (loading) return <div className="p-4">Loading identity...</div>;
  if (config && !canManage) return <Navigate to="/" replace />;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Identity</h2>
          <p className="text-sm text-muted-foreground">Manage bundled ZITADEL users and ArtificialFlow role assignments.</p>
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
          <CardDescription>This screen is available only in bundled ZITADEL mode for artificialflow admin users.</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          <Badge>Bundled ZITADEL</Badge>
          <Badge variant="success">{ARTIFICIALFLOW_ADMIN_ROLE}</Badge>
          <Badge variant="outline">{identity?.principal?.email || identity?.principal?.subject}</Badge>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Add user</CardTitle>
          <CardDescription>Create a human ZITADEL user and assign ArtificialFlow roles.</CardDescription>
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
              <RolePicker platformRoles={platformRoleKeys} customRoles={customRoleKeys} selected={newUser.roles} onToggle={(role) => setNewUser({ ...newUser, roles: roleToggle(newUser.roles, role) })} />
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
              <div className="md:col-span-2"><RolePicker platformRoles={platformRoleKeys} customRoles={customRoleKeys} selected={editingUser.roles || []} onToggle={(role) => setEditingUser({ ...editingUser, roles: roleToggle(editingUser.roles || [], role) })} /></div>
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
                    {isInactiveUserState(user.state) ? (
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={saving}
                        title="Activate user"
                        onClick={() => mutate(() => api.activateIdentityManagementUser(user.id))}
                      >
                        <UserCheck className="mr-2 h-4 w-4" />
                        Activate
                      </Button>
                    ) : (
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={saving}
                        title="Make user inactive"
                        onClick={() => mutate(() => api.terminateIdentityManagementUser(user.id))}
                      >
                        <UserX className="mr-2 h-4 w-4" />
                        Make inactive
                      </Button>
                    )}
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
        <CardHeader>
          <CardTitle>Roles</CardTitle>
          <CardDescription>
            ArtificialFlow admin and ArtificialFlow modeler are static human roles. ArtificialFlow client is reserved for SDK clients.
            Add custom business roles here and enroll users to them.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <form className="grid gap-3 md:grid-cols-4" onSubmit={submitRole}>
            <Input
              required
              placeholder="Role key, e.g. accountant"
              value={newRole.key}
              onChange={(event) => setNewRole({ ...newRole, key: event.target.value })}
            />
            <Input
              required
              placeholder="Display name, e.g. Accountant"
              value={newRole.display_name}
              onChange={(event) => setNewRole({ ...newRole, display_name: event.target.value })}
            />
            <Input
              placeholder="Group, e.g. Finance"
              value={newRole.group}
              onChange={(event) => setNewRole({ ...newRole, group: event.target.value })}
            />
            <Button type="submit" disabled={saving}>
              <Plus className="mr-2 h-4 w-4" />
              Add role
            </Button>
          </form>
          {editingRoleKey && <form className="grid gap-3 md:grid-cols-4" onSubmit={submitRoleEdit}><Input value={editingRoleKey} disabled /><Input required value={editingRole.display_name} onChange={(event) => setEditingRole({ ...editingRole, display_name: event.target.value })} /><Input value={editingRole.group} onChange={(event) => setEditingRole({ ...editingRole, group: event.target.value })} /><div className="flex gap-2"><Button type="submit" disabled={saving}>Save</Button><Button type="button" variant="outline" onClick={() => setEditingRoleKey(null)}>Cancel</Button></div></form>}
          <Table>
            <TableHeader><TableRow><TableHead>Key</TableHead><TableHead>Display name</TableHead><TableHead>Group</TableHead><TableHead className="text-right">Actions</TableHead></TableRow></TableHeader>
            <TableBody>
              {roles.map((role) => {
                const platformRole = isPlatformRole(role.key);
                const deprecatedViewerRole = role.key.toLowerCase() === FLOWGO_VIEWER_ROLE;
                return (
                  <TableRow key={role.key}>
                    <TableCell className="font-mono">
                      <div className="flex flex-wrap items-center gap-2">
                        {role.key}
                        {canonicalizeRole(role.key) === ARTIFICIALFLOW_ADMIN_ROLE && <Badge variant="success">Administrative</Badge>}
                        {canonicalizeRole(role.key) === ARTIFICIALFLOW_MODELER_ROLE && <Badge variant="outline">Modeler</Badge>}
                        {canonicalizeRole(role.key) === ARTIFICIALFLOW_CLIENT_ROLE && (
                          <Badge variant="outline">SDK client</Badge>
                        )}
                        {deprecatedViewerRole && (
                          <Badge variant="warning">Deprecated</Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>{role.display_name}</TableCell>
                    <TableCell>{role.group || "-"}</TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={platformRole}
                          title={platformRole ? "Built-in ArtificialFlow roles cannot be edited here" : "Edit role"}
                          onClick={() => {
                            setEditingRoleKey(role.key);
                            setEditingRole({ display_name: role.display_name, group: role.group });
                          }}
                        >
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="destructive"
                          size="sm"
                          disabled={saving || platformRole}
                          title={platformRole ? "Built-in ArtificialFlow roles cannot be deleted" : "Delete role"}
                          onClick={() =>
                            window.confirm(`Delete role ${role.key}?`) &&
                            mutate(() => api.deleteIdentityManagementRole(role.key))
                          }
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
              {roles.length === 0 && <TableRow><TableCell colSpan={4} className="text-center text-sm text-muted-foreground">No roles found</TableCell></TableRow>}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}

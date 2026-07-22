import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { api, type WorkflowInstance } from "@/lib/api";
import { isAdmin } from "@/lib/roles";
import {
  consumePendingInstanceSync,
  waitForInstanceInList,
  waitForInstanceRemovedFromList,
} from "@/lib/cqrsSync";
import { Eye, Trash2, RefreshCw } from "lucide-react";

export default function Instances() {
  const [instances, setInstances] = useState<WorkflowInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [syncNotice, setSyncNotice] = useState<string | null>(null);
  const [syncingInstanceId, setSyncingInstanceId] = useState<string | null>(null);
  const [canDeleteInstances, setCanDeleteInstances] = useState(false);

  const fetchInstances = useCallback(async (showLoading = true) => {
    if (showLoading) {
      setLoading(true);
    }
    try {
      const data = await api.getInstances();
      setInstances((data || []).filter((instance) => instance.status === "PENDING" || instance.status === "RUNNING"));
      setError(null);
    } catch (err) {
      console.error(err);
      setError("Failed to load instances");
    } finally {
      setLoading(false);
    }
  }, []);

  const syncInstanceIntoList = useCallback(async (id: string) => {
    setSyncingInstanceId(id);
    setSyncNotice("Syncing new instance from the query projection...");
    const result = await waitForInstanceInList(id);
    await fetchInstances(false);
    setSyncNotice(
      result === "synced"
        ? "Instance list is up to date."
        : "Instance started. Instance list is still syncing; use Refresh to check again.",
    );
    setSyncingInstanceId(null);
  }, [fetchInstances]);

  useEffect(() => {
    fetchInstances();
    api.getIdentity().then((identity) => {
      setCanDeleteInstances(isAdmin(identity));
    }).catch(() => {
      setCanDeleteInstances(false);
    });
    const pendingInstanceId = consumePendingInstanceSync();
    if (pendingInstanceId) {
      void syncInstanceIntoList(pendingInstanceId);
    }
  }, [fetchInstances, syncInstanceIntoList]);

  const handleDeleteInstance = async (id: string) => {
    if (!window.confirm("Are you sure you want to delete this instance?")) {
      return;
    }
    try {
        await api.deleteInstance(id);
        setSyncingInstanceId(id);
        setSyncNotice("Delete accepted. Waiting for the instance list to sync...");
        const result = await waitForInstanceRemovedFromList(id);
        await fetchInstances(false);
        setSyncNotice(
          result === "synced"
            ? "Instance list is up to date."
            : "Delete succeeded. Instance list is still syncing; use Refresh to check again.",
        );
    } catch (err) {
        console.error("Failed to delete instance:", err);
        alert("Failed to delete instance");
    } finally {
        setSyncingInstanceId(null);
    }
  };

  if (loading) {
    return <div className="p-4">Loading instances...</div>;
  }

  if (error) {
    return <div className="p-4 text-red-500">{error}</div>;
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Instances</h2>
          <p className="text-sm text-muted-foreground">
            In-progress instances only. Completed instances are available on the History screen.
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => fetchInstances()} disabled={loading || Boolean(syncingInstanceId)}>
          <RefreshCw className={`mr-2 h-4 w-4 ${loading || syncingInstanceId ? "animate-spin" : ""}`} />
          Refresh
        </Button>
      </div>
      {syncNotice ? (
        <div className="rounded-md border bg-muted p-3 text-sm text-muted-foreground">
          {syncNotice}
        </div>
      ) : null}
      <div className="rounded-md border bg-card text-card-foreground">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>Workflow ID</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Context</TableHead>
              <TableHead className="text-right">Created At</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {instances.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center text-muted-foreground">
                  No in-progress instances found. Completed instances are available on the History screen.
                </TableCell>
              </TableRow>
            ) : (
              instances.map((instance) => (
                <TableRow key={instance.id}>
                  <TableCell className="font-medium">{instance.id}</TableCell>
                  <TableCell>{instance.workflow_id}</TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        instance.status === "COMPLETED"
                          ? "success"
                          : instance.status === "FAILED"
                          ? "destructive"
                          : "default"
                      }
                    >
                      {instance.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground max-w-[200px] truncate">
                    {JSON.stringify(instance.context)}
                  </TableCell>
                  <TableCell className="text-right">
                    {new Date(instance.created_at).toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end space-x-2">
                        <Button variant="ghost" size="sm" asChild disabled={syncingInstanceId === instance.id}>
                        <Link to={`/instances/${instance.id}`}>
                            <Eye className="h-4 w-4" />
                        </Link>
                        </Button>
                        {canDeleteInstances ? (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDeleteInstance(instance.id)}
                            disabled={syncingInstanceId === instance.id}
                          >
                              <Trash2 className="h-4 w-4 text-destructive" />
                              {syncingInstanceId === instance.id ? <span className="ml-2">Syncing...</span> : null}
                          </Button>
                        ) : null}
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}

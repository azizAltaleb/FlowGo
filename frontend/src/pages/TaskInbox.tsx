import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router";
import { api, type UserTask, type WorkflowInstance } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Inbox, RefreshCw } from "lucide-react";

type InboxRow = {
  instance: WorkflowInstance;
  tasks: UserTask[];
};

export default function TaskInbox() {
  const [rows, setRows] = useState<InboxRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyKey, setBusyKey] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const instances = await api.listInboxInstances();
      const withTasks = await Promise.all(
        instances.map(async (instance) => {
          try {
            const tasks = await api.listInboxTasks(String(instance.id));
            return { instance, tasks };
          } catch {
            return { instance, tasks: [] as UserTask[] };
          }
        }),
      );
      setRows(withTasks.filter((row) => row.tasks.some((t) => t.canClaim || t.canComplete || t.state !== "COMPLETED")));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load inbox");
      setRows([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const claim = async (instanceId: string, executionId: string) => {
    setBusyKey(`${instanceId}:${executionId}:claim`);
    try {
      await api.claimInboxTask(instanceId, executionId);
      await load();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Claim failed");
    } finally {
      setBusyKey(null);
    }
  };

  const complete = async (instanceId: string, executionId: string) => {
    setBusyKey(`${instanceId}:${executionId}:complete`);
    try {
      await api.completeInboxTask(instanceId, executionId);
      await load();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Complete failed");
    } finally {
      setBusyKey(null);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Inbox className="h-6 w-6" />
          <div>
            <h1 className="text-2xl font-bold">Task Inbox</h1>
            <p className="text-sm text-muted-foreground">
              Claim and complete human tasks assigned to you or your groups.
            </p>
          </div>
        </div>
        <Button variant="outline" size="sm" onClick={() => void load()} disabled={loading}>
          <RefreshCw className={`mr-2 h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          Refresh
        </Button>
      </div>

      {error && <div className="rounded-md border border-destructive/40 bg-destructive/5 p-3 text-sm text-destructive">{error}</div>}
      {loading && <div className="text-sm text-muted-foreground">Loading inbox…</div>}
      {!loading && rows.length === 0 && !error && (
        <Card>
          <CardContent className="py-10 text-center text-sm text-muted-foreground">
            No open tasks in your inbox.
          </CardContent>
        </Card>
      )}

      <div className="space-y-4">
        {rows.map(({ instance, tasks }) => (
          <Card key={String(instance.id)}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-base">
                Instance{" "}
                <Link className="underline" to={`/instances/${instance.id}`}>
                  {instance.id}
                </Link>
              </CardTitle>
              <Badge variant="outline">{instance.status}</Badge>
            </CardHeader>
            <CardContent className="space-y-3">
              {tasks.map((task) => (
                <div key={task.executionId} className="flex items-center justify-between rounded-md border p-3">
                  <div className="space-y-1 text-sm">
                    <div className="font-medium">{task.elementId}</div>
                    <div className="text-xs text-muted-foreground">
                      State {task.state}
                      {task.claimedBy ? ` · claimed by ${task.claimedBy}` : ""}
                      {task.dueDate ? ` · due ${new Date(task.dueDate).toLocaleString()}` : ""}
                    </div>
                  </div>
                  <div className="flex gap-2">
                    {task.canClaim && (
                      <Button
                        size="sm"
                        variant="outline"
                        disabled={busyKey === `${instance.id}:${task.executionId}:claim`}
                        onClick={() => void claim(String(instance.id), task.executionId)}
                      >
                        Take
                      </Button>
                    )}
                    {task.canComplete && (
                      <Button
                        size="sm"
                        disabled={busyKey === `${instance.id}:${task.executionId}:complete`}
                        onClick={() => void complete(String(instance.id), task.executionId)}
                      >
                        Complete
                      </Button>
                    )}
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}

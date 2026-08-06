import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { api, type UserTask, type WorkflowInstance } from "@/lib/api";
import { buildProcessLookup, resolveProcessRef, type ProcessRef } from "@/lib/processLookup";
import { Eye, RefreshCw } from "lucide-react";
import VariablesEditor from "@/components/VariablesEditor";

type CompletedInstanceHistory = {
  instance: WorkflowInstance;
  actions: UserTask[];
};

function formatDate(value?: string): string {
  if (!value) return "Unknown";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function actorForTask(task: UserTask): string {
  return task.claimedBy || task.assignee || "Unknown actor";
}

export default function History() {
  const [rows, setRows] = useState<CompletedInstanceHistory[]>([]);
  const [processLookup, setProcessLookup] = useState<Map<string, ProcessRef>>(new Map());
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHistory = useCallback(async (showLoading = true) => {
    if (showLoading) {
      setLoading(true);
    } else {
      setRefreshing(true);
    }

    try {
      const [completed, workflows] = await Promise.all([
        api.getCompletedInstanceHistory(),
        api.getWorkflows().catch(() => []),
      ]);
      setProcessLookup(buildProcessLookup(workflows || []));
      const historyRows = await Promise.all(
        completed.map(async (instance) => {
          try {
            const actions = await api.listUserTasks(instance.id, { includeCompleted: true });
            return {
              instance,
              actions: actions.filter((task) => task.state === "COMPLETED"),
            };
          } catch (err) {
            console.error("Failed to load task history", { instanceId: instance.id, err });
            return { instance, actions: [] };
          }
        }),
      );
      setRows(historyRows);
      setError(null);
    } catch (err) {
      console.error("Failed to load completed instance history", err);
      setError("Failed to load completed instance history");
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    void fetchHistory();
  }, [fetchHistory]);

  if (loading) {
    return <div className="p-4">Loading completed instance history...</div>;
  }

  if (error) {
    return <div className="p-4 text-red-500">{error}</div>;
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Completed Instance History</h2>
          <p className="text-sm text-muted-foreground">
            Review completed instances, user-task actions, and who took each action.
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => fetchHistory(false)} disabled={refreshing}>
          <RefreshCw className={`mr-2 h-4 w-4 ${refreshing ? "animate-spin" : ""}`} />
          Refresh
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Completed Instances</CardTitle>
          <CardDescription>Action history is loaded from completed user-task jobs.</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Instance ID</TableHead>
                <TableHead>Process Name</TableHead>
                <TableHead>Definition ID</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Completed At</TableHead>
                <TableHead className="min-w-[320px]">Variables</TableHead>
                <TableHead>Actions Taken</TableHead>
                <TableHead className="text-right">View</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center text-muted-foreground">
                    No completed instances found.
                  </TableCell>
                </TableRow>
              ) : (
                rows.map(({ instance, actions }) => {
                  const process = resolveProcessRef(processLookup, instance.workflow_id);
                  return (
                  <TableRow key={instance.id}>
                    <TableCell className="font-medium font-mono text-xs">{instance.id}</TableCell>
                    <TableCell className="font-medium">{process.processName}</TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">{process.definitionKey}</TableCell>
                    <TableCell>
                      <Badge variant="success">{instance.status}</Badge>
                    </TableCell>
                    <TableCell>{formatDate(instance.updated_at || instance.created_at)}</TableCell>
                    <TableCell className="align-top">
                      <VariablesEditor value={instance.context || {}} compact />
                    </TableCell>
                    <TableCell>
                      {actions.length === 0 ? (
                        <span className="text-sm text-muted-foreground">No recorded user task actions</span>
                      ) : (
                        <div className="space-y-2">
                          {actions.map((action) => (
                            <div key={action.key} className="rounded-md border bg-muted/40 p-2 text-sm">
                              <div className="font-medium">{action.elementId}</div>
                              <div className="text-muted-foreground">
                                Completed by {actorForTask(action)} at {formatDate(action.updatedAt)}
                              </div>
                              <div className="mt-1 flex flex-wrap gap-1">
                                {action.assignee ? <Badge variant="outline">Assignee: {action.assignee}</Badge> : null}
                                {(action.candidateUsers || []).map((user) => (
                                  <Badge key={user} variant="outline">User: {user}</Badge>
                                ))}
                                {(action.candidateGroups || []).map((group) => (
                                  <Badge key={group} variant="outline">Group: {group}</Badge>
                                ))}
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button variant="ghost" size="sm" asChild>
                        <Link to={`/instances/${instance.id}`}>
                          <Eye className="h-4 w-4" />
                        </Link>
                      </Button>
                    </TableCell>
                  </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}

import { useEffect, useState, useCallback } from "react";
import { useParams, useNavigate } from "react-router";
import { api, type IdentityResponse, type UserTask, type WorkflowInstance } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { ArrowLeft, Save, RefreshCw } from "lucide-react";
import BpmnViewer from "@/components/bpmn/BpmnViewer";
import { isAdmin } from "@/lib/roles";

export default function InstanceDetails() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [instance, setInstance] = useState<WorkflowInstance | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [variables, setVariables] = useState<string>("");
  const [saving, setSaving] = useState(false);
  const [bpmnXml, setBpmnXml] = useState<string>("");
  const [identity, setIdentity] = useState<IdentityResponse | null>(null);

  const fetchInstance = useCallback(async (isRefresh = false) => {
    if (!id) return;
    
    if (isRefresh) {
        setRefreshing(true);
    } else {
        setLoading(true);
    }

    try {
      const data = await api.getInstance(id);
      setInstance(data);
      // Only update variables text if it's the initial load or user hasn't modified it? 
      // For now, always update to show latest server state, assuming user knows.
      // Actually, if user is typing, we shouldn't overwrite. 
      // But this is a simple refresh. Let's overwrite.
      setVariables(JSON.stringify(data.context, null, 2));

      // Fetch workflow definition for XML
      try {
          const currentIdentity = await api.getIdentity();
          setIdentity(currentIdentity);
      } catch (identityErr) {
          console.error("Failed to load identity:", identityErr);
      }

      try {
          const workflow = await api.getWorkflow(data.workflow_id);
          if (workflow.bpmn_xml) {
              setBpmnXml(workflow.bpmn_xml);
          }
      } catch (wfErr) {
          console.error("Failed to load workflow definition:", wfErr);
      }

    } catch (err) {
      console.error(err);
      setError("Failed to load instance details");
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [id]);

  useEffect(() => {
    fetchInstance();
  }, [fetchInstance]);

  const handleCompleteTask = async (stepId: string) => {
    if (!id) return;
    try {
        await api.completeTask(id, stepId);
        // Refresh data
        const data = await api.getInstance(id);
        setInstance(data);
        setVariables(JSON.stringify(data.context, null, 2));
    } catch (err) {
        console.error("Failed to complete task:", err);
        alert("Failed to complete task");
    }
  };

  const handleClaimUserTask = async (executionId: string) => {
    if (!id) return;
    try {
      await api.claimUserTask(id, executionId);
      const data = await api.getInstance(id);
      setInstance(data);
      setVariables(JSON.stringify(data.context, null, 2));
    } catch (err) {
      console.error("Failed to take task:", err);
      alert(err instanceof Error ? err.message : "Failed to take task");
    }
  };

  const handleCompleteUserTask = async (executionId: string) => {
    if (!id) return;
    try {
      await api.completeUserTask(id, executionId);
      const data = await api.getInstance(id);
      setInstance(data);
      setVariables(JSON.stringify(data.context, null, 2));
    } catch (err) {
      console.error("Failed to complete user task:", err);
      alert(err instanceof Error ? err.message : "Failed to complete user task");
    }
  };

  const handleSaveVariables = async () => {
    if (!id) return;
    setSaving(true);
    try {
      const parsedVariables = JSON.parse(variables);
      await api.updateInstanceVariables(id, parsedVariables);
      // Refresh data
      const data = await api.getInstance(id);
      setInstance(data);
      setVariables(JSON.stringify(data.context, null, 2));
      alert("Variables updated successfully");
    } catch (err) {
      console.error(err);
      alert("Failed to update variables. Ensure valid JSON.");
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <div className="p-8">Loading...</div>;
  if (error || !instance) return <div className="p-8 text-red-500">{error || "Instance not found"}</div>;

  const canManageTasks = isAdmin(identity);

  return (
    <div className="space-y-6">
      <div className="flex items-center space-x-4">
        <Button variant="ghost" size="icon" onClick={() => navigate("/instances")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="text-2xl font-bold">Instance {instance.id}</h1>
        <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => fetchInstance(true)} disabled={loading || refreshing}>
                <RefreshCw className={`mr-2 h-4 w-4 ${refreshing ? "animate-spin" : ""}`} />
                Refresh
            </Button>
        </div>
      </div>

      {/* BPMN Visualization */}
      {bpmnXml && (
        <Card className="h-[500px] flex flex-col">
            <CardHeader className="py-3 px-4 border-b flex flex-row items-center justify-between">
                <CardTitle className="text-base">Process Visualization</CardTitle>
                <div className="flex gap-2">
                   <Badge variant="outline" className="text-blue-600 border-blue-200 bg-blue-50">Active</Badge>
                   <Badge variant="outline" className="text-green-600 border-green-200 bg-green-50">Completed</Badge>
                </div>
            </CardHeader>
            <div className="flex-1 bg-gray-50">
                <BpmnViewer 
                  xml={bpmnXml} 
                  activeStepId={instance.current_step} 
                  executions={instance.executions}
                />
            </div>
        </Card>
      )}

      <div className="grid gap-6 md:grid-cols-2">
        <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>Details</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-2 text-sm">
                  <div className="font-medium">ID:</div>
                  <div>{instance.id}</div>
                  
                  <div className="font-medium">Workflow ID:</div>
                  <div>{instance.workflow_id}</div>
                  
                  <div className="font-medium">Status:</div>
                  <div>
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
                  </div>
                  
                  <div className="font-medium">Current Step(s):</div>
                  <div>
                    {instance.executions
                      .filter((e) => e.status === "ACTIVE")
                      .map((e) => e.step_id)
                      .join(", ") || "None"}
                  </div>
                  
                  <div className="font-medium">Created At:</div>
                  <div>{new Date(instance.created_at).toLocaleString()}</div>
                  
                  <div className="font-medium">Updated At:</div>
                  <div>{new Date(instance.updated_at).toLocaleString()}</div>
                </div>
              </CardContent>
            </Card>

            <Card>
                <CardHeader>
                    <CardTitle>Active Tasks</CardTitle>
                    <div className="text-xs text-muted-foreground">
                      Signed in as {identity?.principal?.email || identity?.principal?.name || identity?.principal?.subject || "unknown user"}
                    </div>
                </CardHeader>
                <CardContent>
                    {instance.executions.filter(e => e.status === 'ACTIVE').length === 0 ? (
                        <div className="text-sm text-muted-foreground">No active tasks.</div>
                    ) : (
                        <div className="space-y-3">
                            {instance.executions.filter(e => e.status === 'ACTIVE').map(exec => (
                                <div key={exec.id} className="flex items-center justify-between p-3 border rounded-md bg-white">
                                    <div className="space-y-1">
                                        <div className="text-sm font-medium">{exec.step_id}</div>
                                        <div className="text-xs text-muted-foreground">Started: {new Date(exec.start_time).toLocaleTimeString()}</div>
                                        <div className="text-[10px] text-gray-400">ID: {exec.id}</div>
                                        {exec.task ? <TaskAssignment task={exec.task} /> : null}
                                    </div>
                                    {exec.task ? (
                                      <div className="flex flex-col items-end gap-2">
                                        {canManageTasks && exec.task.canComplete ? (
                                          <Button size="sm" onClick={() => handleCompleteUserTask(exec.id)}>
                                            Complete
                                          </Button>
                                        ) : canManageTasks && exec.task.canClaim ? (
                                          <Button size="sm" variant="outline" onClick={() => handleClaimUserTask(exec.id)}>
                                            Take Task
                                          </Button>
                                        ) : exec.task.canClaim || exec.task.canComplete ? (
                                          <Button size="sm" disabled>
                                            Use Transaction Inbox
                                          </Button>
                                        ) : (
                                          <Button size="sm" disabled>
                                            Not Assigned
                                          </Button>
                                        )}
                                        {exec.task.claimedBy ? (
                                          <div className="text-[10px] text-muted-foreground">Claimed by {exec.task.claimedBy}</div>
                                        ) : null}
                                      </div>
                                    ) : canManageTasks ? (
                                      <Button size="sm" onClick={() => handleCompleteTask(exec.id)}>
                                          Complete
                                      </Button>
                                    ) : (
                                      <Button size="sm" disabled>
                                        Waiting for worker
                                      </Button>
                                    )}
                                </div>
                            ))}
                        </div>
                    )}
                </CardContent>
            </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Variables (Context)</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>JSON Editor</Label>
              <Textarea
                className="font-mono h-[300px]"
                value={variables}
                readOnly={!canManageTasks}
                onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setVariables(e.target.value)}
              />
            </div>
            {canManageTasks ? (
              <Button onClick={handleSaveVariables} disabled={saving}>
                <Save className="mr-2 h-4 w-4" />
                {saving ? "Saving..." : "Update Variables"}
              </Button>
            ) : (
              <div className="text-sm text-muted-foreground">
                Variables are read-only in the web console for business roles.
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function TaskAssignment({ task }: { task: UserTask }) {
  const candidateGroups = task.candidateGroups || [];
  const candidateUsers = task.candidateUsers || [];
  return (
    <div className="space-y-1">
      <div className="flex flex-wrap gap-1">
        {task.assignee ? <Badge variant="outline">Assignee: {task.assignee}</Badge> : null}
        {candidateGroups.map((group) => (
          <Badge key={group} variant="secondary">Group: {group}</Badge>
        ))}
        {candidateUsers.map((user) => (
          <Badge key={user} variant="secondary">User: {user}</Badge>
        ))}
        {!task.assignee && candidateGroups.length === 0 && candidateUsers.length === 0 ? (
          <Badge variant="outline">Open task</Badge>
        ) : null}
      </div>
    </div>
  );
}

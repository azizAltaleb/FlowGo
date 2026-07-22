import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api, type WorkflowDefinition } from "@/lib/api";
import {
  consumePendingWorkflowSync,
  setPendingInstanceSync,
  waitForWorkflowInCatalog,
  waitForWorkflowRemovedFromCatalog,
} from "@/lib/cqrsSync";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Modal } from "@/components/ui/modal";
import { Play, Edit, Plus, Trash2, RefreshCw } from "lucide-react";
import { isAdmin } from "@/lib/roles";

export default function Processes() {
  const navigate = useNavigate();
  const [workflows, setWorkflows] = useState<WorkflowDefinition[]>([]);
  const [canRunProcesses, setCanRunProcesses] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [syncNotice, setSyncNotice] = useState<string | null>(null);
  const [syncingWorkflowId, setSyncingWorkflowId] = useState<string | null>(null);
  const [syncingCatalog, setSyncingCatalog] = useState(false);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [newProcessName, setNewProcessName] = useState("");
  const [newProcessID, setNewProcessID] = useState("");

  const fetchWorkflows = useCallback(async (showLoading = true) => {
    if (showLoading) {
      setLoading(true);
    }
    setError(null);
    try {
      const data = await api.getWorkflows();
      setWorkflows(data || []);
      setError(null);
    } catch (error) {
      console.error("Failed to fetch workflows:", error);
      setError("Failed to load workflows");
    } finally {
      setLoading(false);
    }
  }, []);

  const syncWorkflowCatalog = useCallback(async (workflowId: string) => {
    setSyncingCatalog(true);
    setSyncNotice("Syncing process catalog from the query projection...");
    const result = await waitForWorkflowInCatalog(workflowId);
    await fetchWorkflows(false);
    setSyncNotice(
      result === "synced"
        ? "Process catalog is up to date."
        : "Command succeeded. Process catalog is still syncing; use Refresh to check again.",
    );
    setSyncingCatalog(false);
  }, [fetchWorkflows]);

  useEffect(() => {
    fetchWorkflows();
    api.getIdentity().then((identity) => {
      setCanRunProcesses(isAdmin(identity));
    }).catch(() => {
      setCanRunProcesses(false);
    });
    const pendingWorkflowId = consumePendingWorkflowSync();
    if (pendingWorkflowId) {
      void syncWorkflowCatalog(pendingWorkflowId);
    }
  }, [fetchWorkflows, syncWorkflowCatalog]);

  const handleStartInstance = async (workflowId: string) => {
    try {
      const instance = await api.startInstance(workflowId);
      setPendingInstanceSync(instance.id);
      alert(`Instance ${instance.id} started. It may take a few seconds to appear on Instances.`);
    } catch (error) {
      console.error("Failed to start instance:", error);
      alert("Failed to start instance");
    }
  };

  const handleDeleteWorkflow = async (workflowId: string) => {
    if (!window.confirm("Are you sure you want to delete this workflow? This cannot be undone.")) {
      return;
    }
    try {
      await api.deleteWorkflow(workflowId);
      setSyncingWorkflowId(workflowId);
      setSyncNotice("Delete accepted. Waiting for the process catalog to sync...");
      const result = await waitForWorkflowRemovedFromCatalog(workflowId);
      await fetchWorkflows(false);
      setSyncNotice(
        result === "synced"
          ? "Process catalog is up to date."
          : "Delete succeeded. Process catalog is still syncing; use Refresh to check again.",
      );
    } catch (error) {
      console.error("Failed to delete workflow:", error);
      alert("Failed to delete workflow");
    } finally {
      setSyncingWorkflowId(null);
    }
  };

  const handleCreateWorkflow = () => {
    if (!newProcessName || !newProcessID) {
      alert("Please fill in all fields");
      return;
    }
    // Navigate to modeler with query params to initialize new diagram
    navigate(
      `/modeler?new=true&name=${encodeURIComponent(
        newProcessName
      )}&id=${encodeURIComponent(newProcessID)}`
    );
  };

  const handleEditWorkflow = (id: string) => {
    navigate(`/modeler?id=${id}`);
  };

  if (loading) {
    return <div>Loading...</div>;
  }

  if (error) {
    return (
      <div className="space-y-4">
        <div className="flex justify-between items-center">
          <h1 className="text-2xl font-bold">Processes</h1>
          <Button variant="outline" size="sm" onClick={() => fetchWorkflows()} disabled={loading}>
            <RefreshCw className={`mr-2 h-4 w-4 ${loading ? "animate-spin" : ""}`} />
            Retry
          </Button>
        </div>
        <div className="rounded-md border border-destructive/40 bg-destructive/10 p-4 text-sm text-destructive">
          {error}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">Processes</h1>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => fetchWorkflows()} disabled={loading || syncingCatalog}>
            <RefreshCw className={`mr-2 h-4 w-4 ${loading || syncingCatalog ? "animate-spin" : ""}`} />
            Refresh
          </Button>
          <Button onClick={() => setIsCreateModalOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Create Workflow
          </Button>
        </div>
      </div>

      {syncNotice ? (
        <div className="rounded-md border bg-muted p-3 text-sm text-muted-foreground">
          {syncNotice}
        </div>
      ) : null}

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>Name</TableHead>
              <TableHead>Version</TableHead>
              <TableHead>Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {workflows.length === 0 ? (
              <TableRow>
                <TableCell colSpan={4} className="text-center text-muted-foreground">
                  No workflows found. Create one to get started.
                </TableCell>
              </TableRow>
            ) : (
              workflows.map((workflow) => (
                <TableRow key={workflow.id}>
                <TableCell>{workflow.process_definition_id}</TableCell>
                <TableCell>{workflow.name}</TableCell>
                <TableCell>{workflow.version}</TableCell>
                <TableCell className="space-x-2">
                  {canRunProcesses ? (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleStartInstance(workflow.id.toString())}
                      disabled={syncingWorkflowId === workflow.id.toString()}
                    >
                      <Play className="mr-2 h-4 w-4" />
                      Start
                    </Button>
                  ) : null}
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleEditWorkflow(workflow.id.toString())}
                    disabled={syncingWorkflowId === workflow.id.toString()}
                  >
                    <Edit className="mr-2 h-4 w-4" />
                    View/Edit
                  </Button>
                  {canRunProcesses ? (
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => handleDeleteWorkflow(workflow.id.toString())}
                      disabled={syncingWorkflowId === workflow.id.toString()}
                    >
                      <Trash2 className="mr-2 h-4 w-4" />
                      {syncingWorkflowId === workflow.id.toString() ? "Syncing..." : "Delete"}
                    </Button>
                  ) : null}
                </TableCell>
              </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <Modal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        title="Create New Workflow"
      >
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="process-name">Process Name</Label>
            <Input
              id="process-name"
              placeholder="e.g. Order Processing"
              value={newProcessName}
              onChange={(e) => setNewProcessName(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="process-id">Process ID (Technical)</Label>
            <Input
              id="process-id"
              placeholder="e.g. Process_Order"
              value={newProcessID}
              onChange={(e) => setNewProcessID(e.target.value)}
            />
          </div>
          <div className="flex justify-end space-x-2 pt-4">
            <Button
              variant="outline"
              onClick={() => setIsCreateModalOpen(false)}
            >
              Cancel
            </Button>
            <Button onClick={handleCreateWorkflow}>Create</Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}

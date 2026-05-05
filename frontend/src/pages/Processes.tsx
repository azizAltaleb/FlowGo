import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api, type WorkflowDefinition } from "@/lib/api";
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
import { Play, Edit, Plus, Trash2 } from "lucide-react";

export default function Processes() {
  const navigate = useNavigate();
  const [workflows, setWorkflows] = useState<WorkflowDefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [newProcessName, setNewProcessName] = useState("");
  const [newProcessID, setNewProcessID] = useState("");

  const fetchWorkflows = async () => {
    try {
      const data = await api.getWorkflows();
      setWorkflows(data || []);
    } catch (error) {
      console.error("Failed to fetch workflows:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchWorkflows();
  }, []);

  const handleStartInstance = async (workflowId: string) => {
    try {
      await api.startInstance(workflowId);
      alert("Instance started successfully");
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
      setWorkflows(workflows.filter(w => w.id !== workflowId));
    } catch (error) {
      console.error("Failed to delete workflow:", error);
      alert("Failed to delete workflow");
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

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">Processes</h1>
        <Button onClick={() => setIsCreateModalOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create Workflow
        </Button>
      </div>

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
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleStartInstance(workflow.id.toString())}
                  >
                    <Play className="mr-2 h-4 w-4" />
                    Start
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleEditWorkflow(workflow.id.toString())}
                  >
                    <Edit className="mr-2 h-4 w-4" />
                    View/Edit
                  </Button>
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => handleDeleteWorkflow(workflow.id.toString())}
                  >
                    <Trash2 className="mr-2 h-4 w-4" />
                    Delete
                  </Button>
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

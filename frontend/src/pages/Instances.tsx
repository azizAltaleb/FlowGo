import { useEffect, useState } from "react";
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
import { Eye, Trash2, RefreshCw } from "lucide-react";

export default function Instances() {
  const [instances, setInstances] = useState<WorkflowInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchInstances = async () => {
    setLoading(true);
    try {
      const data = await api.getInstances();
      setInstances(data || []);
      setError(null);
    } catch (err) {
      console.error(err);
      setError("Failed to load instances");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchInstances();
  }, []);

  const handleDeleteInstance = async (id: string) => {
    if (!window.confirm("Are you sure you want to delete this instance?")) {
      return;
    }
    try {
        await api.deleteInstance(id);
        setInstances(instances.filter(i => i.id !== id));
    } catch (err) {
        console.error("Failed to delete instance:", err);
        alert("Failed to delete instance");
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
        <h2 className="text-2xl font-bold tracking-tight">Instances</h2>
        <Button variant="outline" size="sm" onClick={fetchInstances} disabled={loading}>
          <RefreshCw className={`mr-2 h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          Refresh
        </Button>
      </div>
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
                  No active instances found.
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
                        <Button variant="ghost" size="sm" asChild>
                        <Link to={`/instances/${instance.id}`}>
                            <Eye className="h-4 w-4" />
                        </Link>
                        </Button>
                        <Button variant="ghost" size="sm" onClick={() => handleDeleteInstance(instance.id)}>
                            <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
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

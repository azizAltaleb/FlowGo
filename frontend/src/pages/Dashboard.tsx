import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { OverviewChart } from "@/components/OverviewChart";
import { Activity, CheckCircle, Clock, XCircle, RefreshCw } from "lucide-react";
import { api, type WorkflowInstance } from "@/lib/api";

export default function Dashboard() {
  const [instances, setInstances] = useState<WorkflowInstance[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchData = async () => {
    setLoading(true);
    try {
      const data = await api.getInstances();
      setInstances(data);
    } catch (err) {
      console.error("Failed to load dashboard data", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  // Calculate stats from instances
  const total = instances.length;
  const active = instances.filter((i) => i.status === "RUNNING").length;
  const completed = instances.filter((i) => i.status === "COMPLETED").length;
  const failed = instances.filter((i) => i.status === "FAILED").length;

  // Calculate chart data (instances by month)
  const chartData = [
    { name: "Jan", total: 0 },
    { name: "Feb", total: 0 },
    { name: "Mar", total: 0 },
    { name: "Apr", total: 0 },
    { name: "May", total: 0 },
    { name: "Jun", total: 0 },
    { name: "Jul", total: 0 },
    { name: "Aug", total: 0 },
    { name: "Sep", total: 0 },
    { name: "Oct", total: 0 },
    { name: "Nov", total: 0 },
    { name: "Dec", total: 0 },
  ];

  instances.forEach((instance) => {
    const date = new Date(instance.created_at);
    const month = date.getMonth(); // 0-11
    if (month >= 0 && month < 12) {
      chartData[month].total += 1;
    }
  });

  const stats = [
    {
      title: "Total Instances",
      value: total.toString(),
      icon: Activity,
      color: "text-blue-500",
    },
    {
      title: "Active",
      value: active.toString(),
      icon: Clock,
      color: "text-yellow-500",
    },
    {
      title: "Completed",
      value: completed.toString(),
      icon: CheckCircle,
      color: "text-green-500",
    },
    {
      title: "Failed",
      value: failed.toString(),
      icon: XCircle,
      color: "text-red-500",
    },
  ];

  if (loading) {
    return <div className="p-4">Loading dashboard...</div>;
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h2 className="text-3xl font-bold tracking-tight">Dashboard</h2>
        <Button variant="outline" size="sm" onClick={fetchData} disabled={loading}>
          <RefreshCw className={`mr-2 h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          Refresh
        </Button>
      </div>
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => {
          const Icon = stat.icon;
          return (
            <Card key={stat.title}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  {stat.title}
                </CardTitle>
                <Icon className={`h-4 w-4 ${stat.color}`} />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{stat.value}</div>
                <p className="text-xs text-muted-foreground">
                  Real-time data
                </p>
              </CardContent>
            </Card>
          );
        })}
      </div>
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-7">
        <Card className="col-span-4">
          <CardHeader>
            <CardTitle>Overview</CardTitle>
          </CardHeader>
          <CardContent className="pl-2">
            <OverviewChart data={chartData} />
          </CardContent>
        </Card>
        <Card className="col-span-3">
          <CardHeader>
            <CardTitle>Recent Activity</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-8">
              {instances.slice(0, 5).map((i) => (
                <div key={i.id} className="flex items-center">
                  <div className="space-y-1">
                    <p className="text-sm font-medium leading-none">
                      Instance #{i.id} {i.status.toLowerCase()}
                    </p>
                    <p className="text-sm text-muted-foreground">
                      Workflow: {i.workflow_id}
                    </p>
                  </div>
                  <div className="ml-auto font-medium text-xs text-muted-foreground">
                    {new Date(i.created_at).toLocaleTimeString()}
                  </div>
                </div>
              ))}
              {instances.length === 0 && (
                <p className="text-sm text-muted-foreground">No recent activity.</p>
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

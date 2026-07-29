import { useEffect, useState } from "react";
import { Link } from "react-router";
import { api } from "@/lib/api";
import { readQueryProjectionSampledAt } from "@/lib/cqrsStatus";

const STALE_MS = 15_000;

/**
 * Shows query-projection freshness for admins. Relies on list fetches updating
 * session timestamps via cqrsSync / page loads.
 */
export function CqrsStatusBanner({ admin }: { admin: boolean }) {
  const [lastSyncAt, setLastSyncAt] = useState<number | null>(null);
  const [outboxPending, setOutboxPending] = useState<number | null>(null);

  useEffect(() => {
    const read = () => setLastSyncAt(readQueryProjectionSampledAt());
    read();
    const id = window.setInterval(read, 2000);
    return () => window.clearInterval(id);
  }, []);

  useEffect(() => {
    if (!admin) return;
    let cancelled = false;
    const loadMetrics = async () => {
      try {
        const metrics = await api.getEngineMetrics();
        if (!cancelled) {
          setOutboxPending(typeof metrics.outboxPending === "number" ? metrics.outboxPending : null);
        }
      } catch {
        if (!cancelled) setOutboxPending(null);
      }
    };
    void loadMetrics();
    const id = window.setInterval(loadMetrics, 10000);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [admin]);

  const ageMs = lastSyncAt ? Date.now() - lastSyncAt : null;
  const stale = ageMs !== null && ageMs > STALE_MS;

  return (
    <div className="flex items-center gap-3 text-xs text-muted-foreground">
      <span>
        Projection:{" "}
        {lastSyncAt
          ? stale
            ? `stale (${Math.round((ageMs || 0) / 1000)}s)`
            : `fresh (${Math.round((ageMs || 0) / 1000)}s ago)`
          : "not sampled yet"}
      </span>
      {admin && outboxPending !== null && <span>Outbox pending: {outboxPending}</span>}
      {admin && (
        <Link className="underline" to="/?rebuild=1">
          Rebuild guidance
        </Link>
      )}
    </div>
  );
}

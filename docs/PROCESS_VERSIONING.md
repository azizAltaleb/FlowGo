# Process versioning and migration

## How versions work

Each deploy of a BPMN process id creates a new `Process` row with an incremented `version` and a unique definition key.

- **Start by definition key** (`workflow_id` numeric): always starts that exact version.
- **Start by BPMN process id** (string): starts the **latest** version unless `version` is supplied.

```json
POST /instances
{
  "workflow_id": "golden_order_approval",
  "version": 2,
  "context": { "orderId": "…" }
}
```

## Open instances (1.0 policy)

ArtificialFlow does **not** silently migrate tokens between definition versions.

| Policy | Behavior |
| :--- | :--- |
| Drain | Leave running instances on their original definition; start new work on the new version. |
| Migrate | Not automatic. Remodel + restart, or custom ops scripts. |

Recommended production default: **drain**. Document cutover dates for long-running processes.

## UI

The Processes page lists each deployed version and starts that row’s definition key (pinned version).

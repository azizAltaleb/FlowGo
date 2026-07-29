# ArtificialFlow Python worker SDK

Minimal Python client for the ArtificialFlow worker protocol (`/jobs/*`).

## Install

```bash
pip install -e clients/python-sdk
```

## Example

```python
from artificialflow_worker import Worker

def handle(job):
    return {"ok": True, "elementId": job.get("elementId")}

Worker(
    base_url="http://localhost:9100/api",
    token="...",
    job_type="golden-validate",
    worker_name="python-sample",
    handler=handle,
).run()
```

Requires Python 3.10+.

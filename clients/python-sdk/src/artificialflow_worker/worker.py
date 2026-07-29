"""Minimal ArtificialFlow worker client (activate / complete / fail)."""

from __future__ import annotations

import json
import time
import uuid
from typing import Any, Callable, Optional
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

Handler = Callable[[dict[str, Any]], dict[str, Any]]


class Worker:
    def __init__(
        self,
        *,
        base_url: str,
        token: str,
        job_type: str,
        worker_name: str,
        handler: Handler,
        max_jobs: int = 1,
        activate_timeout_ms: int = 5000,
        lock_duration_ms: int = 30000,
        poll_interval_sec: float = 1.0,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self.token = token
        self.job_type = job_type
        self.worker_name = worker_name
        self.handler = handler
        self.max_jobs = max_jobs
        self.activate_timeout_ms = activate_timeout_ms
        self.lock_duration_ms = lock_duration_ms
        self.poll_interval_sec = poll_interval_sec

    def _request(self, method: str, path: str, body: Optional[dict[str, Any]] = None) -> Any:
        data = None if body is None else json.dumps(body).encode("utf-8")
        req = Request(
            f"{self.base_url}{path}",
            data=data,
            method=method,
            headers={
                "Authorization": f"Bearer {self.token}",
                "Content-Type": "application/json",
                "X-Workflow-Worker-Protocol-Version": "v1",
                "Idempotency-Key": str(uuid.uuid4()),
            },
        )
        with urlopen(req, timeout=60) as resp:
            raw = resp.read().decode("utf-8")
            if not raw:
                return None
            try:
                return json.loads(raw)
            except json.JSONDecodeError:
                return raw

    def run(self) -> None:
        while True:
            try:
                activated = self._request(
                    "POST",
                    "/jobs/activate",
                    {
                        "type": self.job_type,
                        "worker": self.worker_name,
                        "maxJobs": self.max_jobs,
                        "timeout": self.activate_timeout_ms,
                        "lockDurationMs": self.lock_duration_ms,
                    },
                )
                jobs = (activated or {}).get("jobs") or []
                if not jobs:
                    time.sleep(self.poll_interval_sec)
                    continue
                for job in jobs:
                    key = job.get("key")
                    try:
                        variables = self.handler(job) or {}
                        self._request(
                            "POST",
                            f"/jobs/{key}/complete",
                            {"worker": self.worker_name, "variables": variables},
                        )
                    except Exception as exc:  # noqa: BLE001
                        self._request(
                            "POST",
                            f"/jobs/{key}/fail",
                            {
                                "worker": self.worker_name,
                                "errorMessage": str(exc),
                                "retries": max(int(job.get("retries") or 1) - 1, 0),
                            },
                        )
            except (HTTPError, URLError, TimeoutError):
                time.sleep(self.poll_interval_sec)

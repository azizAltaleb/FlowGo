# Worker API

Workers use the command service job APIs to activate, complete, fail, and extend locks for external jobs.

## Compatibility

- Protocol negotiation request header: `X-Workflow-Worker-Protocol-Version`.
- Protocol negotiation response header: `X-Workflow-Engine-Protocol-Version`.
- Capabilities endpoint: `GET /jobs/capabilities`.

## Idempotency

Mutation calls should include an `Idempotency-Key` header:

- Complete job.
- Fail job.
- Extend lock.

Retries with the same key should be replay-safe when the original request was accepted.

## Conformance

Run:

```bash
make worker-conformance
```

The Node.js SDK should preserve wire compatibility with this API.

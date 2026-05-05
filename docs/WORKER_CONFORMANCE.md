# Worker Conformance

The worker conformance smoke test validates the external worker API contract.

Run:

```bash
make worker-conformance
```

The test checks protocol negotiation, capabilities, and mutation behavior for worker integrations.

SDK and API changes that affect `/jobs/*` should update this guide and the conformance smoke test.

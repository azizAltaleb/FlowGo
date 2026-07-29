# Decision tables (DMN-lite)

ArtificialFlow evaluates JSON decision tables referenced by `artificialflow:decisionRef` on business-rule tasks.

## UI upload

Open **Decisions** in the ArtificialFlow console (admin or modeler). Upload a JSON file or paste the decision table, then **Deploy**. Use **Evaluate** to dry-run inputs without starting a process.

## Deploy (API)

```bash
curl -X POST "$API/decisions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "decision_id": "invoice_decision",
    "name": "Invoice routing",
    "resource": "{\"id\":\"invoice_decision\",\"hitPolicy\":\"FIRST\",\"rules\":[{\"when\":{\"amount\":{\"op\":\"gt\",\"value\":1000}},\"then\":{\"result\":\"manager\"}},{\"when\":{},\"then\":{\"result\":\"auto\"}}]}"
  }'
```

```bash
curl -H "Authorization: Bearer $TOKEN" "$API/decisions"
```

## Evaluate

```bash
curl -X POST "$API/decisions/invoice_decision/evaluate" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"inputs":{"amount":1500}}'
```

Outputs are written to the instance context (`result_variable`, default `decisionResult`) and flattened into top-level keys.

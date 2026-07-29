package dmn

import "testing"

func TestEvaluateFirstHit(t *testing.T) {
	table, err := Parse([]byte(`{
		"id": "invoice_decision",
		"hitPolicy": "FIRST",
		"rules": [
			{"when": {"amount": {"op": "gt", "value": 1000}}, "then": {"result": "manager"}},
			{"when": {}, "then": {"result": "auto"}}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := Evaluate(table, map[string]any{"amount": 1500})
	if err != nil {
		t.Fatal(err)
	}
	if out["result"] != "manager" {
		t.Fatalf("got %#v", out)
	}
	out2, err := Evaluate(table, map[string]any{"amount": 10})
	if err != nil {
		t.Fatal(err)
	}
	if out2["result"] != "auto" {
		t.Fatalf("got %#v", out2)
	}
}

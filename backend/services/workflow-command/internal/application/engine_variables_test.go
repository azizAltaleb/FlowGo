package application

import "testing"

func TestSnapshotVariablesCopiesMap(t *testing.T) {
	original := map[string]any{
		"approved": false,
		"count":    1,
	}

	snapshot := snapshotVariables(original)
	original["approved"] = true

	if snapshot["approved"] != false {
		t.Fatalf("expected snapshot to keep original value, got %v", snapshot["approved"])
	}
	if snapshot["count"] != 1 {
		t.Fatalf("expected snapshot to include count, got %v", snapshot["count"])
	}
}

func TestChangedVariablesReturnsOnlyChangedKeys(t *testing.T) {
	before := map[string]any{
		"unchanged": "keep",
		"changed":   1,
		"nested":    map[string]any{"value": "same"},
	}
	after := map[string]any{
		"unchanged": "keep",
		"changed":   2,
		"nested":    map[string]any{"value": "same"},
		"new":       true,
	}

	changed := changedVariables(before, after)
	if len(changed) != 2 {
		t.Fatalf("expected 2 changed variables, got %d: %v", len(changed), changed)
	}
	if changed["changed"] != 2 {
		t.Fatalf("expected changed value 2, got %v", changed["changed"])
	}
	if changed["new"] != true {
		t.Fatalf("expected new value true, got %v", changed["new"])
	}
	if _, ok := changed["unchanged"]; ok {
		t.Fatalf("did not expect unchanged key to be returned")
	}
	if _, ok := changed["nested"]; ok {
		t.Fatalf("did not expect deeply equal nested key to be returned")
	}
}

func TestChangedVariablesReturnsNilForNoChanges(t *testing.T) {
	before := map[string]any{"approved": true}
	after := map[string]any{"approved": true}

	if changed := changedVariables(before, after); changed != nil {
		t.Fatalf("expected no changed variables, got %v", changed)
	}
}

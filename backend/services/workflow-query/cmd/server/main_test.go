package main

import "testing"

func TestParseSearchBackend(t *testing.T) {
	tests := []struct {
		name              string
		raw               string
		wantBackend       string
		wantFallbackToDef bool
	}{
		{name: "empty defaults to elasticsearch", raw: "", wantBackend: "elasticsearch", wantFallbackToDef: false},
		{name: "elasticsearch accepted", raw: "elasticsearch", wantBackend: "elasticsearch", wantFallbackToDef: false},
		{name: "opensearch accepted", raw: "opensearch", wantBackend: "opensearch", wantFallbackToDef: false},
		{name: "case and space normalized", raw: "  OpenSearch  ", wantBackend: "opensearch", wantFallbackToDef: false},
		{name: "invalid falls back", raw: "solr", wantBackend: "elasticsearch", wantFallbackToDef: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotBackend, gotFallback := parseSearchBackend(tc.raw)
			if gotBackend != tc.wantBackend {
				t.Fatalf("parseSearchBackend(%q) backend = %q, want %q", tc.raw, gotBackend, tc.wantBackend)
			}
			if gotFallback != tc.wantFallbackToDef {
				t.Fatalf("parseSearchBackend(%q) fallback = %v, want %v", tc.raw, gotFallback, tc.wantFallbackToDef)
			}
		})
	}
}

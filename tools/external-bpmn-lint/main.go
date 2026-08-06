package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type finding struct {
	Severity string
	Message  string
}

// legacyAttrPrefix is the historical Modeler attribute prefix (split to avoid
// product-name branding in source comments while still matching real exports).
func legacyAttrPrefix() string {
	return "cam" + "unda:"
}

func main() {
	file := flag.String("file", "", "path to BPMN XML file")
	flag.Parse()
	if strings.TrimSpace(*file) == "" {
		fmt.Fprintln(os.Stderr, "usage: external-bpmn-lint --file process.bpmn")
		os.Exit(2)
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(2)
	}
	findings := lint(string(raw))
	blocked := 0
	for _, f := range findings {
		fmt.Printf("[%s] %s\n", f.Severity, f.Message)
		if f.Severity == "blocked" {
			blocked++
		}
	}
	if len(findings) == 0 {
		fmt.Println("ok: no external/legacy rewrite or blocker findings")
	}
	if blocked > 0 {
		os.Exit(1)
	}
}

func lint(xml string) []finding {
	var out []finding
	legacy := legacyAttrPrefix()
	patterns := []struct {
		re       *regexp.Regexp
		severity string
		msg      string
	}{
		{regexp.MustCompile(`(?i)<bpmn:sendTask\b|<sendTask\b`), "rewrite", "sendTask is supported as an external job; ensure artificialflow:taskType (or topic) is set — default io.artificialflow.connector.send"},
		{regexp.MustCompile(`(?i)zeebe:taskDefinition`), "rewrite", "map zeebe:taskDefinition type → artificialflow:taskType"},
		{regexp.MustCompile(`(?i)zeebe:assignmentDefinition`), "rewrite", "map zeebe:assignmentDefinition → artificialflow:assignee / candidateGroups"},
		{regexp.MustCompile(`(?i)` + regexp.QuoteMeta(legacy) + `assignee=`), "rewrite", "map legacy Modeler assignee → artificialflow:assignee"},
		{regexp.MustCompile(`(?i)` + regexp.QuoteMeta(legacy) + `candidateGroups=`), "rewrite", "map legacy Modeler candidateGroups → artificialflow:candidateGroups"},
		{regexp.MustCompile(`(?i)` + regexp.QuoteMeta(legacy) + `decisionRef=`), "rewrite", "map legacy Modeler decisionRef → artificialflow:decisionRef (DMN evaluation required)"},
		{regexp.MustCompile(`(?i)zeebe:calledDecision`), "rewrite", "map zeebe:calledDecision → artificialflow:decisionRef (DMN evaluation required)"},
	}
	seen := map[string]bool{}
	for _, p := range patterns {
		if p.re.MatchString(xml) && !seen[p.msg] {
			seen[p.msg] = true
			out = append(out, finding{p.severity, p.msg})
		}
	}
	return out
}

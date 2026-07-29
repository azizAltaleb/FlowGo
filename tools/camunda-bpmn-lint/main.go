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

func main() {
	file := flag.String("file", "", "path to BPMN XML file")
	flag.Parse()
	if strings.TrimSpace(*file) == "" {
		fmt.Fprintln(os.Stderr, "usage: camunda-bpmn-lint --file process.bpmn")
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
		fmt.Println("ok: no Camunda/Zeebe rewrite or blocker findings")
	}
	if blocked > 0 {
		os.Exit(1)
	}
}

func lint(xml string) []finding {
	var out []finding
	if regexp.MustCompile(`(?i)<bpmn:sendTask\b|<sendTask\b`).MatchString(xml) {
		out = append(out, finding{"blocked", "sendTask is not supported; remodel as serviceTask + worker or message throw"})
	}
	patterns := []struct {
		re      *regexp.Regexp
		severity string
		msg     string
	}{
		{regexp.MustCompile(`(?i)zeebe:taskDefinition`), "rewrite", "map zeebe:taskDefinition type → artificialflow:taskType"},
		{regexp.MustCompile(`(?i)zeebe:assignmentDefinition`), "rewrite", "map zeebe:assignmentDefinition → artificialflow:assignee / candidateGroups"},
		{regexp.MustCompile(`(?i)camunda:assignee=`), "rewrite", "map camunda:assignee → artificialflow:assignee"},
		{regexp.MustCompile(`(?i)camunda:candidateGroups=`), "rewrite", "map camunda:candidateGroups → artificialflow:candidateGroups"},
		{regexp.MustCompile(`(?i)camunda:decisionRef=`), "rewrite", "map camunda:decisionRef → artificialflow:decisionRef (DMN evaluation required)"},
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

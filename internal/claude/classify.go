package claude

import (
	"path/filepath"
	"slices"
	"strings"
)

type knowledgeFieldRule struct {
	Label    string
	Keywords []string
}

var knowledgeFieldRules = []knowledgeFieldRule{
	{Label: "Frontend", Keywords: []string{"ui", "ux", "css", "html", "svelte", "react", "component", "layout", "design", "threejs", "three.js", "canvas", "svg", "browser"}},
	{Label: "Backend", Keywords: []string{"api", "server", "handler", "route", "endpoint", "middleware", "http", "rpc", "auth", "backend"}},
	{Label: "Data", Keywords: []string{"database", "db", "sql", "schema", "query", "migration", "json", "parser", "graph", "index"}},
	{Label: "Docs", Keywords: []string{"readme", "docs", "documentation", "guide", "markdown", "md", "comment", "spec"}},
	{Label: "Testing", Keywords: []string{"test", "tests", "testing", "assert", "fixture", "smoke", "integration", "regression", "bug"}},
	{Label: "Infra", Keywords: []string{"docker", "deploy", "deployment", "ci", "build", "config", "env", "runtime", "kubernetes", "terraform"}},
	{Label: "Agents", Keywords: []string{"claude", "agent", "subagent", "prompt", "memory", "mcp", "tool_use", "reasoning", "assistant"}},
}

func classifyKnowledgeFields(texts []string, toolNames []string, pathRefs []string) []string {
	var corpus []string
	corpus = append(corpus, texts...)
	corpus = append(corpus, toolNames...)
	for _, pathRef := range pathRefs {
		corpus = append(corpus, pathRef, filepath.Base(pathRef))
	}

	lower := strings.ToLower(strings.Join(corpus, "\n"))
	if strings.TrimSpace(lower) == "" {
		return []string{"General"}
	}

	var fields []string
	for _, rule := range knowledgeFieldRules {
		if matchesKnowledgeField(lower, rule.Keywords) {
			fields = append(fields, rule.Label)
		}
	}

	if len(fields) == 0 {
		fields = append(fields, "General")
	}
	slices.Sort(fields)
	return fields
}

func normalizeKnowledgeFields(fields []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, field := range fields {
		if field == "" {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		out = append(out, field)
	}
	if len(out) > 1 {
		filtered := out[:0]
		for _, field := range out {
			if field == "General" {
				continue
			}
			filtered = append(filtered, field)
		}
		out = filtered
	}
	slices.Sort(out)
	if len(out) == 0 {
		return []string{"General"}
	}
	return out
}

func matchesKnowledgeField(corpus string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(corpus, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

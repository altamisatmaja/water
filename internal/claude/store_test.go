package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreLoadProjectByQuery(t *testing.T) {
	root := t.TempDir()
	projectKey := "-Users-demo-letbarqris"
	projectDir := filepath.Join(root, projectKey)
	if err := os.MkdirAll(filepath.Join(projectDir, "memory"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "session-1", "subagents"), 0o755); err != nil {
		t.Fatal(err)
	}

	session := `{"type":"user","message":{"role":"user","content":"update README and CLAUDE"},"uuid":"u1","timestamp":"2026-03-31T11:22:25Z","cwd":"/Users/demo/letbarqris","sessionId":"session-1","gitBranch":"main"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/Users/demo/letbarqris/README.md"}},{"type":"tool_use","name":"Glob","input":{"path":"/Users/demo/letbarqris/docs"}}],"usage":{"input_tokens":10,"output_tokens":4}},"uuid":"a1","timestamp":"2026-03-31T11:22:28Z","cwd":"/Users/demo/letbarqris","sessionId":"session-1","gitBranch":"main"}`
	if err := os.WriteFile(filepath.Join(projectDir, "session-1.jsonl"), []byte(session), 0o644); err != nil {
		t.Fatal(err)
	}

	meta := `{"agentType":"Explore","description":"Explore docs"}`
	if err := os.WriteFile(filepath.Join(projectDir, "session-1", "subagents", "agent-a1.meta.json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}

	subagent := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"docs/guide.md"}}]},"cwd":"/Users/demo/letbarqris","sessionId":"session-1"}`
	if err := os.WriteFile(filepath.Join(projectDir, "session-1", "subagents", "agent-a1.jsonl"), []byte(subagent), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(projectDir, "memory", "MEMORY.md"), []byte("# Memory Index\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}

	snapshot, err := store.LoadProjectByQuery("letbarqris", "")
	if err != nil {
		t.Fatal(err)
	}

	if snapshot.Project.Key != projectKey {
		t.Fatalf("expected project key %q, got %q", projectKey, snapshot.Project.Key)
	}
	if snapshot.Stats.Sessions != 1 {
		t.Fatalf("expected 1 session, got %d", snapshot.Stats.Sessions)
	}
	if snapshot.Stats.MemoryFiles != 1 {
		t.Fatalf("expected 1 memory file, got %d", snapshot.Stats.MemoryFiles)
	}
	if len(snapshot.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(snapshot.Tools))
	}
	if len(snapshot.Paths) != 3 {
		t.Fatalf("expected 3 paths, got %d", len(snapshot.Paths))
	}
	if len(snapshot.Fields) == 0 {
		t.Fatal("expected knowledge fields to be classified from session JSONL")
	}
	if len(snapshot.Sessions[0].KnowledgeFields) == 0 {
		t.Fatal("expected session knowledge fields to be populated")
	}
	var hasFieldNode, hasPathNode bool
	for _, node := range snapshot.Graph.Nodes {
		if node.Kind == "field" {
			hasFieldNode = true
		}
		if node.Kind == "path" || node.Kind == "file" || node.Kind == "dir" {
			hasPathNode = true
		}
	}
	if !hasFieldNode {
		t.Fatal("expected graph to include field nodes")
	}
	if !hasPathNode {
		t.Fatal("expected graph to keep path/file nodes")
	}
	if len(snapshot.Graph.Nodes) == 0 || len(snapshot.Graph.Edges) == 0 {
		t.Fatal("expected graph nodes and edges to be populated")
	}
}

func TestEncodeProjectPath(t *testing.T) {
	got := EncodeProjectPath("/Users/demo/project")
	want := "-Users-demo-project"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

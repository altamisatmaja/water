package claude

import (
	"os"
	"testing"
)

func TestLocalClaudeSmoke(t *testing.T) {
	if os.Getenv("WATER_CLAUDE_SMOKE") != "1" {
		t.Skip("set WATER_CLAUDE_SMOKE=1 to run against local ~/.claude data")
	}

	query := os.Getenv("WATER_CLAUDE_PROJECT_QUERY")
	if query == "" {
		t.Skip("set WATER_CLAUDE_PROJECT_QUERY to a local Claude project name or path")
	}

	store, err := NewStore(os.Getenv("WATER_CLAUDE_PROJECTS_PATH"))
	if err != nil {
		t.Fatal(err)
	}

	snapshot, err := store.LoadProjectByQuery(query, "")
	if err != nil {
		t.Fatal(err)
	}

	if snapshot.Project == nil {
		t.Fatal("expected project metadata")
	}
	if snapshot.Stats.Sessions == 0 {
		t.Fatal("expected at least one session in local Claude project")
	}
	if snapshot.Graph == nil || len(snapshot.Graph.Nodes) == 0 {
		t.Fatal("expected graph nodes from local Claude project")
	}
}

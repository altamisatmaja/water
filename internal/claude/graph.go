package claude

import (
	"fmt"
	"path/filepath"
	"slices"
)

func buildGraph(ref *ProjectRef, sessions []*SessionSummary, memory []*MemoryFile, paths []*PathSummary, tools []*ToolSummary, fields []*KnowledgeFieldSummary) *Graph {
	graph := &Graph{}
	projectNodeID := "project:" + ref.Key
	graph.Nodes = append(graph.Nodes, &GraphNode{
		ID:      projectNodeID,
		Label:   ref.Name,
		Kind:    "project",
		Group:   "project",
		Size:    16,
		Path:    ref.Path,
		Preview: ref.CWD,
	})

	for _, memoryFile := range memory {
		nodeID := "memory:" + memoryFile.RelativePath
		graph.Nodes = append(graph.Nodes, &GraphNode{
			ID:      nodeID,
			Label:   memoryFile.Name,
			Kind:    "memory",
			Group:   "memory",
			Size:    8,
			Path:    memoryFile.Path,
			Preview: memoryFile.Preview,
		})
		graph.Edges = append(graph.Edges, &GraphEdge{
			Source: projectNodeID,
			Target: nodeID,
			Kind:   "memorizes",
			Weight: 1,
		})
	}

	for _, field := range fields {
		fieldNodeID := "field:" + field.Label
		graph.Nodes = append(graph.Nodes, &GraphNode{
			ID:             fieldNodeID,
			Label:          field.Label,
			Kind:           "field",
			Group:          "field",
			Size:           max(7, min(16, 7+field.MessageCount/12)),
			RefCount:       field.MessageCount,
			Preview:        field.Description,
			KnowledgeField: field.Label,
		})
		graph.Edges = append(graph.Edges, &GraphEdge{
			Source: projectNodeID,
			Target: fieldNodeID,
			Kind:   "classifies",
			Weight: max(1, len(field.Sessions)),
		})
	}

	sessionsCopy := append([]*SessionSummary(nil), sessions...)
	slices.Reverse(sessionsCopy)
	var previousSessionNode string
	for _, session := range sessionsCopy {
		sessionNodeID := "session:" + session.ID
		graph.Nodes = append(graph.Nodes, &GraphNode{
			ID:        sessionNodeID,
			Label:     session.ID[:8],
			Kind:      "session",
			Group:     "session",
			Size:      max(8, min(16, 8+session.MessageCount/12)),
			RefCount:  session.MessageCount,
			Preview:   session.PromptPreview,
			Path:      session.Path,
			SessionID: session.ID,
		})
		graph.Edges = append(graph.Edges, &GraphEdge{
			Source: projectNodeID,
			Target: sessionNodeID,
			Kind:   "contains",
			Weight: max(1, session.MessageCount),
		})

		if previousSessionNode != "" {
			graph.Edges = append(graph.Edges, &GraphEdge{
				Source: previousSessionNode,
				Target: sessionNodeID,
				Kind:   "next",
				Weight: 1,
			})
		}
		previousSessionNode = sessionNodeID

		for _, fieldLabel := range session.KnowledgeFields {
			graph.Edges = append(graph.Edges, &GraphEdge{
				Source: sessionNodeID,
				Target: "field:" + fieldLabel,
				Kind:   "classified_as",
				Weight: max(1, session.MessageCount),
			})
		}

		for _, subagent := range session.Subagents {
			subagentNodeID := "subagent:" + session.ID + ":" + subagent.ID
			label := subagent.AgentType
			if label == "" {
				label = subagent.ID
			}
			graph.Nodes = append(graph.Nodes, &GraphNode{
				ID:        subagentNodeID,
				Label:     label,
				Kind:      "subagent",
				Group:     "subagent",
				Size:      max(6, min(12, 6+subagent.MessageCount/8)),
				RefCount:  subagent.MessageCount,
				Preview:   subagent.Description,
				Path:      subagent.Path,
				SessionID: session.ID,
			})
			graph.Edges = append(graph.Edges, &GraphEdge{
				Source: sessionNodeID,
				Target: subagentNodeID,
				Kind:   "delegates",
				Weight: max(1, subagent.MessageCount),
			})
			for _, fieldLabel := range subagent.KnowledgeFields {
				graph.Edges = append(graph.Edges, &GraphEdge{
					Source: subagentNodeID,
					Target: "field:" + fieldLabel,
					Kind:   "classified_as",
					Weight: max(1, subagent.MessageCount),
				})
			}
		}
	}

	for _, tool := range tools {
		toolNodeID := "tool:" + tool.Name
		graph.Nodes = append(graph.Nodes, &GraphNode{
			ID:       toolNodeID,
			Label:    tool.Name,
			Kind:     "tool",
			Group:    "tool",
			Size:     max(6, min(14, 6+tool.Count)),
			RefCount: tool.Count,
			Preview:  fmt.Sprintf("%d session refs", len(tool.Sessions)),
		})
		for _, sessionID := range tool.Sessions {
			graph.Edges = append(graph.Edges, &GraphEdge{
				Source: "session:" + sessionID,
				Target: toolNodeID,
				Kind:   "uses",
				Weight: tool.Count,
			})
		}
	}

	pathLimit := min(len(paths), 48)
	for _, pathItem := range paths[:pathLimit] {
		nodeID := "path:" + pathItem.Path
		graph.Nodes = append(graph.Nodes, &GraphNode{
			ID:       nodeID,
			Label:    pathItem.Label,
			Kind:     pathItem.Kind,
			Group:    "path",
			Size:     max(6, min(14, 6+pathItem.ReferenceCount/2)),
			RefCount: pathItem.ReferenceCount,
			Preview:  pathItem.Path,
			Path:     pathItem.Path,
		})

		for _, sessionID := range pathItem.Sessions {
			graph.Edges = append(graph.Edges, &GraphEdge{
				Source: "session:" + sessionID,
				Target: nodeID,
				Kind:   "touches",
				Weight: pathItem.ReferenceCount,
			})
		}
	}

	return graph
}

func projectDisplayName(ref *ProjectRef) string {
	if ref == nil {
		return ""
	}
	if ref.CWD != "" {
		return filepath.Base(ref.CWD)
	}
	return ref.Name
}

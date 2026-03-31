package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const scannerMaxTokenSize = 64 * 1024 * 1024

type Store struct {
	projectsRoot string
}

func DefaultProjectsRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".claude", "projects")
	}
	return filepath.Join(home, ".claude", "projects")
}

func NewStore(projectsRoot string) (*Store, error) {
	if projectsRoot == "" {
		projectsRoot = DefaultProjectsRoot()
	}
	projectsRoot = expandHome(projectsRoot)
	info, err := os.Stat(projectsRoot)
	if err != nil {
		return nil, fmt.Errorf("stat claude projects root %s: %w", projectsRoot, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("claude projects root is not a directory: %s", projectsRoot)
	}
	return &Store{projectsRoot: projectsRoot}, nil
}

func (s *Store) ProjectsRoot() string {
	return s.projectsRoot
}

func EncodeProjectPath(path string) string {
	cleaned := filepath.Clean(expandHome(path))
	return strings.ReplaceAll(cleaned, string(filepath.Separator), "-")
}

func (s *Store) ListProjects() ([]*ProjectRef, error) {
	entries, err := os.ReadDir(s.projectsRoot)
	if err != nil {
		return nil, fmt.Errorf("read projects root: %w", err)
	}

	var projects []*ProjectRef
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectPath := filepath.Join(s.projectsRoot, entry.Name())
		ref := &ProjectRef{
			Key:  entry.Name(),
			Name: entry.Name(),
			Path: projectPath,
		}

		if cwd := detectProjectCWD(projectPath); cwd != "" {
			ref.CWD = cwd
			ref.Name = filepath.Base(cwd)
		}

		ref.SessionCount = countSessionFiles(projectPath)
		ref.LastUpdated = latestProjectTimestamp(projectPath)
		projects = append(projects, ref)
	}

	slices.SortFunc(projects, func(a, b *ProjectRef) int {
		if !a.LastUpdated.Equal(b.LastUpdated) {
			if a.LastUpdated.After(b.LastUpdated) {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Name, b.Name)
	})

	return projects, nil
}

func (s *Store) ResolveProject(query, cwd string) (*ProjectRef, error) {
	cwd = expandHome(cwd)
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}

	if ref := s.resolveByPath(query); ref != nil {
		return ref, nil
	}
	if ref := s.resolveByPath(cwd); ref != nil {
		return ref, nil
	}

	projects, err := s.ListProjects()
	if err != nil {
		return nil, err
	}

	query = strings.TrimSpace(query)
	if query == "" {
		query = filepath.Base(cwd)
	}
	if query == "" {
		if len(projects) == 1 {
			return projects[0], nil
		}
		return nil, fmt.Errorf("no claude project query provided")
	}

	q := strings.ToLower(strings.TrimSpace(query))
	type candidate struct {
		score int
		ref   *ProjectRef
	}

	var best *candidate
	for _, project := range projects {
		score := scoreProjectMatch(project, q)
		if score == 0 {
			continue
		}
		if best == nil || score > best.score {
			best = &candidate{score: score, ref: project}
		}
	}

	if best == nil {
		return nil, fmt.Errorf("no Claude project matched %q under %s", query, s.projectsRoot)
	}
	return best.ref, nil
}

func (s *Store) LoadProjectByQuery(query, cwd string) (*ProjectSnapshot, error) {
	ref, err := s.ResolveProject(query, cwd)
	if err != nil {
		return nil, err
	}
	return s.LoadProject(ref)
}

func (s *Store) LoadProject(ref *ProjectRef) (*ProjectSnapshot, error) {
	if ref == nil {
		return nil, fmt.Errorf("load project: nil ref")
	}

	sessions, err := s.loadSessions(ref)
	if err != nil {
		return nil, err
	}
	memory, err := s.loadMemory(ref)
	if err != nil {
		return nil, err
	}

	pathIndex := map[string]*PathSummary{}
	toolIndex := map[string]*ToolSummary{}
	fieldIndex := map[string]*KnowledgeFieldSummary{}
	stats := ProjectStats{
		Sessions:    len(sessions),
		MemoryFiles: len(memory),
	}

	for _, session := range sessions {
		stats.Messages += session.MessageCount
		stats.Subagents += len(session.Subagents)
		stats.InputTokens += session.InputTokens
		stats.OutputTokens += session.OutputTokens

		if session.UpdatedAt.After(ref.LastUpdated) {
			ref.LastUpdated = session.UpdatedAt
		}
		if stats.LatestActivity == "" && !session.UpdatedAt.IsZero() {
			stats.LatestActivity = session.UpdatedAt.Format(time.RFC3339)
		}

		for _, toolName := range session.ToolNames {
			tool := toolIndex[toolName]
			if tool == nil {
				tool = &ToolSummary{Name: toolName}
				toolIndex[toolName] = tool
			}
			tool.Count++
			tool.Sessions = appendUnique(tool.Sessions, session.ID)
		}

		for _, pathRef := range session.PathRefs {
			item := pathIndex[pathRef]
			if item == nil {
				item = &PathSummary{
					Path:  pathRef,
					Label: pathLabel(pathRef),
					Kind:  pathKind(pathRef),
				}
				pathIndex[pathRef] = item
			}
			item.ReferenceCount++
			item.Sessions = appendUnique(item.Sessions, session.ID)
		}

		for _, fieldLabel := range session.KnowledgeFields {
			field := fieldIndex[fieldLabel]
			if field == nil {
				field = &KnowledgeFieldSummary{
					Label:       fieldLabel,
					Description: knowledgeFieldDescription(fieldLabel),
				}
				fieldIndex[fieldLabel] = field
			}
			field.MessageCount += session.MessageCount
			field.Sessions = appendUnique(field.Sessions, session.ID)
		}

		for _, subagent := range session.Subagents {
			for _, toolName := range subagent.ToolNames {
				tool := toolIndex[toolName]
				if tool == nil {
					tool = &ToolSummary{Name: toolName}
					toolIndex[toolName] = tool
				}
				tool.Count++
				tool.Sessions = appendUnique(tool.Sessions, session.ID)
			}

			for _, pathRef := range subagent.PathRefs {
				item := pathIndex[pathRef]
				if item == nil {
					item = &PathSummary{
						Path:  pathRef,
						Label: pathLabel(pathRef),
						Kind:  pathKind(pathRef),
					}
					pathIndex[pathRef] = item
				}
				item.ReferenceCount++
				item.Sessions = appendUnique(item.Sessions, session.ID)
			}

			for _, fieldLabel := range subagent.KnowledgeFields {
				field := fieldIndex[fieldLabel]
				if field == nil {
					field = &KnowledgeFieldSummary{
						Label:       fieldLabel,
						Description: knowledgeFieldDescription(fieldLabel),
					}
					fieldIndex[fieldLabel] = field
				}
				field.MessageCount += subagent.MessageCount
				field.Sessions = appendUnique(field.Sessions, session.ID)
			}
		}
	}

	paths := mapValues(pathIndex)
	slices.SortFunc(paths, func(a, b *PathSummary) int {
		if a.ReferenceCount != b.ReferenceCount {
			return b.ReferenceCount - a.ReferenceCount
		}
		return strings.Compare(a.Path, b.Path)
	})

	tools := mapValues(toolIndex)
	slices.SortFunc(tools, func(a, b *ToolSummary) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		return strings.Compare(a.Name, b.Name)
	})

	fields := mapValues(fieldIndex)
	slices.SortFunc(fields, func(a, b *KnowledgeFieldSummary) int {
		if a.MessageCount != b.MessageCount {
			return b.MessageCount - a.MessageCount
		}
		return strings.Compare(a.Label, b.Label)
	})

	stats.Paths = len(paths)
	stats.Tools = len(tools)
	stats.KnowledgeFields = len(fields)
	graph := buildGraph(ref, sessions, memory, paths, tools, fields)

	return &ProjectSnapshot{
		Project:  ref,
		Stats:    stats,
		Sessions: sessions,
		Memory:   memory,
		Paths:    paths,
		Tools:    tools,
		Fields:   fields,
		Graph:    graph,
	}, nil
}

func (s *Store) resolveByPath(path string) *ProjectRef {
	path = strings.TrimSpace(expandHome(path))
	if path == "" {
		return nil
	}

	for current := filepath.Clean(path); current != string(filepath.Separator) && current != "."; current = filepath.Dir(current) {
		key := EncodeProjectPath(current)
		projectPath := filepath.Join(s.projectsRoot, key)
		info, err := os.Stat(projectPath)
		if err == nil && info.IsDir() {
			return &ProjectRef{
				Key:          key,
				Name:         filepath.Base(current),
				CWD:          current,
				Path:         projectPath,
				SessionCount: countSessionFiles(projectPath),
				LastUpdated:  latestProjectTimestamp(projectPath),
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	return nil
}

func scoreProjectMatch(project *ProjectRef, query string) int {
	fields := []string{
		strings.ToLower(project.Key),
		strings.ToLower(project.Name),
		strings.ToLower(project.CWD),
	}

	score := 0
	for _, field := range fields {
		if field == "" {
			continue
		}
		switch {
		case field == query:
			score = max(score, 100)
		case filepath.Base(field) == query:
			score = max(score, 95)
		case strings.Contains(field, query):
			score = max(score, 75)
		case strings.Contains(strings.ReplaceAll(field, "-", ""), strings.ReplaceAll(query, "-", "")):
			score = max(score, 60)
		}
	}
	return score
}

func detectProjectCWD(projectPath string) string {
	matches, _ := filepath.Glob(filepath.Join(projectPath, "*.jsonl"))
	slices.Sort(matches)
	for _, match := range matches {
		cwd, _ := readFirstSessionCWD(match)
		if cwd != "" {
			return cwd
		}
	}
	return ""
}

func readFirstSessionCWD(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), scannerMaxTokenSize)
	for scanner.Scan() {
		line := scanner.Bytes()
		var item struct {
			CWD string `json:"cwd"`
		}
		if err := json.Unmarshal(line, &item); err == nil && item.CWD != "" {
			return item.CWD, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", nil
}

func countSessionFiles(projectPath string) int {
	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".jsonl") {
			count++
		}
	}
	return count
}

func latestProjectTimestamp(projectPath string) time.Time {
	var latest time.Time
	_ = filepath.WalkDir(projectPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	return latest
}

func expandHome(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

func shorten(text string, limit int) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if limit <= 0 || len(text) <= limit {
		return text
	}
	if limit < 4 {
		return text[:limit]
	}
	return text[:limit-3] + "..."
}

func readAllString(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func appendUnique(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, item := range values {
		if item == value {
			return values
		}
	}
	return append(values, value)
}

func mapValues[T any](items map[string]*T) []*T {
	out := make([]*T, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out
}

func readJSONFile(path string, target any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func knowledgeFieldDescription(label string) string {
	switch label {
	case "Frontend":
		return "UI layers, components, rendering, browser behavior, and interaction design."
	case "Backend":
		return "Server handlers, APIs, routing, middleware, and application services."
	case "Data":
		return "Schemas, parsers, queries, storage structures, and graph/data flow."
	case "Docs":
		return "Documentation, markdown, README work, and written guidance."
	case "Testing":
		return "Test coverage, debugging, smoke checks, and regression work."
	case "Infra":
		return "Build pipelines, deployment, runtime config, and environment setup."
	case "Agents":
		return "Prompts, agent coordination, memory, MCP tools, and assistant workflows."
	default:
		return "General project work without a stronger field signal."
	}
}

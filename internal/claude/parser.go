package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type rawClaudeLine struct {
	Type          string         `json:"type"`
	UUID          string         `json:"uuid"`
	ParentUUID    *string        `json:"parentUuid"`
	IsSidechain   bool           `json:"isSidechain"`
	PromptID      string         `json:"promptId"`
	AgentID       string         `json:"agentId"`
	SessionID     string         `json:"sessionId"`
	Timestamp     time.Time      `json:"timestamp"`
	CWD           string         `json:"cwd"`
	GitBranch     string         `json:"gitBranch"`
	Version       string         `json:"version"`
	Entrypoint    string         `json:"entrypoint"`
	Slug          string         `json:"slug"`
	Message       *rawMessage    `json:"message"`
	ToolUseResult *rawToolResult `json:"toolUseResult"`
}

type rawMessage struct {
	Model   string          `json:"model"`
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	Usage   *rawUsage       `json:"usage"`
}

type rawUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

type rawToolResult struct {
	Type string          `json:"type"`
	File *rawToolFileRef `json:"file"`
}

type rawToolFileRef struct {
	FilePath string `json:"filePath"`
}

type contentSummary struct {
	Preview      string
	ToolNames    []string
	PathRefs     []string
	TextSnippets []string
	UserPrompt   bool
}

func (s *Store) loadSessions(ref *ProjectRef) ([]*SessionSummary, error) {
	entries, err := os.ReadDir(ref.Path)
	if err != nil {
		return nil, fmt.Errorf("read project directory %s: %w", ref.Path, err)
	}

	var sessions []*SessionSummary
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		sessionPath := filepath.Join(ref.Path, entry.Name())
		session, err := s.loadSessionFile(ref, sessionPath)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	slices.SortFunc(sessions, func(a, b *SessionSummary) int {
		if !a.UpdatedAt.Equal(b.UpdatedAt) {
			if a.UpdatedAt.After(b.UpdatedAt) {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})

	return sessions, nil
}

func (s *Store) loadSessionFile(ref *ProjectRef, sessionPath string) (*SessionSummary, error) {
	f, err := os.Open(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("open session %s: %w", sessionPath, err)
	}
	defer f.Close()

	sessionID := strings.TrimSuffix(filepath.Base(sessionPath), ".jsonl")
	session := &SessionSummary{
		ID:   sessionID,
		Path: sessionPath,
	}

	toolSet := map[string]struct{}{}
	pathSet := map[string]struct{}{}
	fieldSet := map[string]struct{}{}
	var classificationTexts []string

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), scannerMaxTokenSize)
	for scanner.Scan() {
		var line rawClaudeLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.SessionID == "" {
			line.SessionID = sessionID
		}

		if session.CWD == "" && line.CWD != "" {
			session.CWD = line.CWD
		}
		if session.GitBranch == "" && line.GitBranch != "" {
			session.GitBranch = line.GitBranch
		}
		if session.Version == "" && line.Version != "" {
			session.Version = line.Version
		}
		if session.Entrypoint == "" && line.Entrypoint != "" {
			session.Entrypoint = line.Entrypoint
		}
		if session.StartedAt.IsZero() && !line.Timestamp.IsZero() {
			session.StartedAt = line.Timestamp
		}
		if line.Timestamp.After(session.UpdatedAt) {
			session.UpdatedAt = line.Timestamp
		}

		if line.Message != nil && line.Message.Usage != nil {
			session.InputTokens += line.Message.Usage.InputTokens
			session.OutputTokens += line.Message.Usage.OutputTokens
		}

		switch line.Type {
		case "user", "assistant":
			session.MessageCount++
			if line.Type == "user" {
				session.UserMessageCount++
			} else {
				session.AssistantMessageCount++
			}
		}

		if line.Message != nil {
			summary := extractContentSummary(line.Message.Content, session.CWD)
			if session.PromptPreview == "" && line.Type == "user" && summary.UserPrompt && summary.Preview != "" {
				session.PromptPreview = summary.Preview
			}
			if line.Type == "assistant" {
				session.ToolUseCount += len(summary.ToolNames)
			}
			for _, toolName := range summary.ToolNames {
				toolSet[toolName] = struct{}{}
			}
			for _, pathRef := range summary.PathRefs {
				pathSet[pathRef] = struct{}{}
			}
			classificationTexts = append(classificationTexts, summary.TextSnippets...)
			for _, field := range classifyKnowledgeFields(summary.TextSnippets, summary.ToolNames, summary.PathRefs) {
				fieldSet[field] = struct{}{}
			}
		}

		if line.ToolUseResult != nil && line.ToolUseResult.File != nil && line.ToolUseResult.File.FilePath != "" {
			pathSet[normalizePath(line.ToolUseResult.File.FilePath, session.CWD)] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan session %s: %w", sessionPath, err)
	}

	session.ToolNames = sortedKeys(toolSet)
	session.PathRefs = sortedKeys(pathSet)
	session.KnowledgeFields = normalizeKnowledgeFields(sortedKeys(fieldSet))

	subagents, err := s.loadSubagents(ref, sessionID, session.CWD)
	if err != nil {
		return nil, err
	}
	session.Subagents = subagents
	if len(session.KnowledgeFields) == 0 {
		session.KnowledgeFields = classifyKnowledgeFields(classificationTexts, session.ToolNames, session.PathRefs)
	}

	return session, nil
}

func (s *Store) loadSubagents(ref *ProjectRef, sessionID, sessionCWD string) ([]*SubagentSummary, error) {
	subagentDir := filepath.Join(ref.Path, sessionID, "subagents")
	info, err := os.Stat(subagentDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat subagents: %w", err)
	}
	if !info.IsDir() {
		return nil, nil
	}

	entries, err := os.ReadDir(subagentDir)
	if err != nil {
		return nil, fmt.Errorf("read subagents: %w", err)
	}

	type agentFiles struct {
		jsonl string
		meta  string
	}
	agentIndex := map[string]*agentFiles{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		base := strings.TrimSuffix(strings.TrimSuffix(name, ".meta.json"), ".jsonl")
		files := agentIndex[base]
		if files == nil {
			files = &agentFiles{}
			agentIndex[base] = files
		}
		switch {
		case strings.HasSuffix(name, ".meta.json"):
			files.meta = filepath.Join(subagentDir, name)
		case strings.HasSuffix(name, ".jsonl"):
			files.jsonl = filepath.Join(subagentDir, name)
		}
	}

	var subagents []*SubagentSummary
	for base, files := range agentIndex {
		subagent := &SubagentSummary{
			ID:   strings.TrimPrefix(base, "agent-"),
			Path: files.jsonl,
		}

		if files.meta != "" {
			var meta struct {
				AgentType   string `json:"agentType"`
				Description string `json:"description"`
			}
			if err := readJSONFile(files.meta, &meta); err == nil {
				subagent.AgentType = meta.AgentType
				subagent.Description = meta.Description
			}
		}

		if files.jsonl != "" {
			if err := populateSubagentSummary(files.jsonl, sessionCWD, subagent); err != nil {
				return nil, err
			}
		}

		subagents = append(subagents, subagent)
	}

	slices.SortFunc(subagents, func(a, b *SubagentSummary) int {
		return strings.Compare(a.ID, b.ID)
	})

	return subagents, nil
}

func populateSubagentSummary(path, sessionCWD string, subagent *SubagentSummary) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open subagent %s: %w", path, err)
	}
	defer f.Close()

	toolSet := map[string]struct{}{}
	pathSet := map[string]struct{}{}
	fieldSet := map[string]struct{}{}
	var classificationTexts []string

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), scannerMaxTokenSize)
	for scanner.Scan() {
		var line rawClaudeLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Message != nil {
			subagent.MessageCount++
			summary := extractContentSummary(line.Message.Content, coalesce(line.CWD, sessionCWD))
			for _, toolName := range summary.ToolNames {
				toolSet[toolName] = struct{}{}
			}
			for _, pathRef := range summary.PathRefs {
				pathSet[pathRef] = struct{}{}
			}
			classificationTexts = append(classificationTexts, summary.TextSnippets...)
			for _, field := range classifyKnowledgeFields(summary.TextSnippets, summary.ToolNames, summary.PathRefs) {
				fieldSet[field] = struct{}{}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan subagent %s: %w", path, err)
	}

	subagent.ToolNames = sortedKeys(toolSet)
	subagent.PathRefs = sortedKeys(pathSet)
	subagent.KnowledgeFields = normalizeKnowledgeFields(sortedKeys(fieldSet))
	if len(subagent.KnowledgeFields) == 0 {
		subagent.KnowledgeFields = classifyKnowledgeFields(classificationTexts, subagent.ToolNames, subagent.PathRefs)
	}
	return nil
}

func (s *Store) loadMemory(ref *ProjectRef) ([]*MemoryFile, error) {
	memoryDir := filepath.Join(ref.Path, "memory")
	info, err := os.Stat(memoryDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat memory dir: %w", err)
	}
	if !info.IsDir() {
		return nil, nil
	}

	var files []*MemoryFile
	err = filepath.WalkDir(memoryDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		content, err := readAllString(path)
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(memoryDir, path)
		files = append(files, &MemoryFile{
			Name:         filepath.Base(path),
			Path:         path,
			RelativePath: rel,
			Preview:      shorten(content, 220),
			Content:      content,
			Size:         info.Size(),
			ModifiedAt:   info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk memory dir: %w", err)
	}

	slices.SortFunc(files, func(a, b *MemoryFile) int {
		return strings.Compare(a.RelativePath, b.RelativePath)
	})
	return files, nil
}

func extractContentSummary(raw json.RawMessage, cwd string) contentSummary {
	var out contentSummary
	if len(raw) == 0 {
		return out
	}

	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return out
	}

	switch value := payload.(type) {
	case string:
		out.Preview = shorten(value, 160)
		out.UserPrompt = strings.TrimSpace(value) != ""
		if text := strings.TrimSpace(value); text != "" {
			out.TextSnippets = append(out.TextSnippets, text)
		}
	case []any:
		var previews []string
		for _, item := range value {
			block, ok := item.(map[string]any)
			if !ok {
				continue
			}
			blockType := asString(block["type"])
			switch blockType {
			case "text":
				text := asString(block["text"])
				if text != "" {
					previews = append(previews, text)
					out.TextSnippets = append(out.TextSnippets, text)
					out.UserPrompt = true
				}
			case "tool_use":
				name := asString(block["name"])
				if name != "" {
					out.ToolNames = appendUnique(out.ToolNames, name)
					previews = append(previews, "tool:"+name)
				}
				for _, pathRef := range collectPaths(block["input"], cwd) {
					out.PathRefs = appendUnique(out.PathRefs, pathRef)
				}
			case "tool_result":
				for _, pathRef := range collectPaths(block, cwd) {
					out.PathRefs = appendUnique(out.PathRefs, pathRef)
				}
				if out.Preview == "" {
					if text := extractTextish(block["content"]); text != "" {
						previews = append(previews, text)
						out.TextSnippets = append(out.TextSnippets, text)
					}
				}
			case "thinking":
				if out.Preview == "" {
					previews = append(previews, "thinking")
				}
			}
		}
		if len(previews) > 0 {
			out.Preview = shorten(strings.Join(previews, " | "), 160)
		}
	case map[string]any:
		if text := extractTextish(value); text != "" {
			out.Preview = shorten(text, 160)
			out.TextSnippets = append(out.TextSnippets, text)
			out.UserPrompt = true
		}
		for _, pathRef := range collectPaths(value, cwd) {
			out.PathRefs = appendUnique(out.PathRefs, pathRef)
		}
	}

	return out
}

func collectPaths(value any, cwd string) []string {
	var paths []string
	switch item := value.(type) {
	case map[string]any:
		for key, nested := range item {
			switch key {
			case "file_path", "path", "cwd":
				if pathValue := asString(nested); pathValue != "" {
					paths = append(paths, normalizePath(pathValue, cwd))
				}
			}
			paths = append(paths, collectPaths(nested, cwd)...)
		}
	case []any:
		for _, nested := range item {
			paths = append(paths, collectPaths(nested, cwd)...)
		}
	}
	return uniqueSortedStrings(paths)
}

func extractTextish(value any) string {
	switch item := value.(type) {
	case string:
		return item
	case []any:
		var parts []string
		for _, nested := range item {
			if text := extractTextish(nested); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, " ")
	case map[string]any:
		if text := asString(item["text"]); text != "" {
			return text
		}
		if content := item["content"]; content != nil {
			return extractTextish(content)
		}
	}
	return ""
}

func normalizePath(pathValue, cwd string) string {
	pathValue = strings.TrimSpace(pathValue)
	if pathValue == "" {
		return ""
	}
	if strings.HasPrefix(pathValue, "~") {
		pathValue = expandHome(pathValue)
	}
	if filepath.IsAbs(pathValue) {
		return filepath.Clean(pathValue)
	}
	if cwd != "" {
		return filepath.Clean(filepath.Join(cwd, pathValue))
	}
	return filepath.Clean(pathValue)
}

func sortedKeys(items map[string]struct{}) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		if key == "" {
			continue
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	slices.Sort(out)
	return out
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	switch item := value.(type) {
	case string:
		return item
	case json.Number:
		return item.String()
	default:
		return fmt.Sprintf("%v", item)
	}
}

func pathLabel(path string) string {
	base := filepath.Base(path)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return path
	}
	return base
}

func pathKind(path string) string {
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return "dir"
		}
		return "file"
	}
	if filepath.Ext(path) != "" {
		return "file"
	}
	return "path"
}

func coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

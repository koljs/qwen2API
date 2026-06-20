package toolcall

import (
	"encoding/json"
	"regexp"
	"strings"
)

func canonicalToolName(name string, allowed map[string]string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	// Try exact match first
	if exact, ok := allowed[strings.ToLower(name)]; ok {
		return exact
	}
	// Try all Qwen alias candidates (primary + alternatives for different IDEs)
	for _, candidate := range qwenToolAliasCandidates(name) {
		if exact, ok := allowed[strings.ToLower(candidate)]; ok {
			return exact
		}
	}
	// Fallback: try key-based matching
	key := toolAliasKey(name)
	for allowedKey, canonical := range allowed {
		if toolAliasKey(allowedKey) == key || toolAliasKey(canonical) == key {
			return canonical
		}
		// Also try matching via alias candidates
		for _, candidate := range qwenToolAliasCandidates(canonical) {
			if toolAliasKey(candidate) == key {
				return canonical
			}
		}
	}
	return ""
}

func toolAliasKey(value string) string {
	return regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "")
}

func qwenToolAlias(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	// Map Qwen tool aliases to their canonical IDE tool names.
	// For tools that have different names across IDEs (e.g. Bash vs RunCommand),
	// we return the most common alias; canonicalToolName will try all candidates.
	explicit := map[string]string{
		"fs_open_file":        "Read",
		"fs_put_file":         "Write",
		"fs_patch_file":       "Edit",
		"fs_delete_file":      "DeleteFile",
		"shell_run":           "RunCommand",
		"check_command_status": "CheckCommandStatus",
		"stop_command":        "StopCommand",
		"text_search":         "Grep",
		"path_find":           "LS",
		"codebase_search":     "SearchCodebase",
		"notebook_patch":      "NotebookEdit",
		"http_get_url":        "WebFetch",
		"web_query":           "WebSearch",
		"task_update":         "TodoWrite",
		"ask_user":            "AskUserQuestion",
		"skill_invoke":        "Skill",
		"delegate_task":       "Task",
		"invoke_skill":        "Skill",
		"task_create":         "Task",
		"cron_create":         "Schedule",
		"schedule_task":       "Schedule",
		"notify_user":         "NotifyUser",
		"open_preview":        "OpenPreview",
	}
	lowered := strings.ToLower(trimmed)
	if mapped, ok := explicit[lowered]; ok {
		return mapped
	}
	trimmedKey := toolAliasKey(trimmed)
	for alias, mapped := range explicit {
		if toolAliasKey(alias) == trimmedKey {
			return mapped
		}
	}
	if strings.HasPrefix(trimmed, "u_") && len(trimmed) > 2 {
		return strings.TrimPrefix(trimmed, "u_")
	}
	return ""
}

// qwenToolAliasCandidates returns all possible canonical names for a Qwen tool alias.
// This handles cases where different IDEs use different tool names for the same
// underlying operation (e.g. Bash vs RunCommand, Glob vs LS).
func qwenToolAliasCandidates(name string) []string {
	candidates := []string{}
	primary := qwenToolAlias(name)
	if primary != "" {
		candidates = append(candidates, primary)
	}
	// Add alternative IDE-specific names for the same tool
	alternatives := map[string][]string{
		"shell_run":      {"Bash", "RunCommand", "execute_command"},
		"path_find":      {"Glob", "LS", "list_directory"},
		"fs_patch_file":  {"Edit", "SearchReplace", "apply_patch"},
		"fs_delete_file": {"DeleteFile", "rm"},
	}
	lowered := strings.ToLower(strings.TrimSpace(name))
	if alts, ok := alternatives[lowered]; ok {
		for _, alt := range alts {
			if alt != primary {
				candidates = append(candidates, alt)
			}
		}
	}
	return candidates
}

func parseToolInput(text string) any {
	if text == "" {
		return map[string]any{}
	}
	var value any
	if err := json.Unmarshal([]byte(text), &value); err == nil {
		return NormalizeToolInput(value)
	}
	re := regexp.MustCompile(`(?is)<([A-Za-z_][A-Za-z0-9_\-]*)>(.*?)</\1>`)
	params := map[string]any{}
	for _, match := range re.FindAllStringSubmatch(text, -1) {
		if len(match) == 3 {
			params[match[1]] = strings.TrimSpace(match[2])
		}
	}
	if len(params) > 0 {
		return params
	}
	if kv := ParseTextKVInput(text); len(kv) > 0 {
		return kv
	}
	return map[string]any{"input": text}
}

// NormalizeToolInput decodes nested JSON strings and supplies empty argument maps.
func NormalizeToolInput(value any) any {
	switch v := value.(type) {
	case nil:
		return map[string]any{}
	case string:
		if strings.TrimSpace(v) == "" {
			return map[string]any{}
		}
		var decoded any
		if err := json.Unmarshal([]byte(v), &decoded); err == nil {
			return NormalizeToolInput(decoded)
		}
		if kv := ParseTextKVInput(v); len(kv) > 0 {
			return kv
		}
		return v
	default:
		return v
	}
}

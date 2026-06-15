package adapter

func traeToolProfile() cliToolProfile {
	return cliToolProfile{
		ID:          "trae",
		DisplayName: "TRAE IDE",
		Match: func(names toolNameSet) bool {
			return names.hasAny("LS", "RunCommand", "SearchCodebase", "AskUserQuestion") &&
				names.hasAny("Read", "Glob", "Grep", "Edit", "Write", "Bash")
		},
		Priority: map[string]int{
			"read":         0,
			"bash":         1,
			"runcommand":   1,
			"glob":         2,
			"ls":           2,
			"grep":         3,
			"searchcodebase": 3,
			"write":        4,
			"edit":         5,
			"webfetch":     6,
			"websearch":    7,
			"todowrite":    80,
			"askuserquestion": 85,
			"task":         40,
			"agent":        40,
			"skill":        50,
			"schedule":     90,
		},
		Rules: []string{
			"When the user asks to list files, explore directories, or view project structure, call path_find with pattern=\"**/*\" or pattern=\"*\" immediately. Do not narrate.",
			"When the user asks to read or view a file, call fs_open_file with file_path set to the file path immediately. Do not narrate.",
			"When the user asks to run a command, execute code, or perform a shell action, call shell_run with command set to the exact command immediately. Do not narrate.",
			"When the user asks to search code or find text in files, call text_search with pattern set to the search query immediately. Do not narrate.",
			"Always emit a <|QNML|tool_calls> block when a tool action is needed. Never describe what you would do instead of calling the tool.",
			"Required parameters must have non-empty values. If the user did not specify a path, use \".\" or the workspace root as a reasonable default.",
		},
	}
}

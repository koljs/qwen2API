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
			"read":            0,
			"bash":            1,
			"runcommand":      1,
			"glob":            2,
			"ls":              2,
			"grep":            3,
			"searchcodebase":  3,
			"write":           4,
			"edit":            5,
			"webfetch":        6,
			"websearch":       7,
			"todowrite":       80,
			"askuserquestion": 85,
			"task":            40,
			"skill":           50,
			"schedule":        90,
		},
		Rules: []string{
			"When a tool action is needed, emit a QNML tool_calls block immediately instead of narrating.",
			"If a required path argument is unspecified, use \".\" as the default workspace root.",
		},
	}
}

package pkg

type ValidationErrors struct {
	ExcludeChecks []string `yaml:"exclude_checks,omitempty"`
}

type DocsSkip struct {
	Title  string `yaml:"title"`
	Reason string `yaml:"reason"`
}

type DocsStructureEnforced struct {
	Enabled bool       `yaml:"enabled"`
	Version int        `yaml:"version"`
	Skip    []DocsSkip `yaml:"skip,omitempty"`
}

type Validation struct {
	Errors ValidationErrors `yaml:"errors,omitempty"`

	DocsStructureEnforced DocsStructureEnforced `yaml:"docs_structure_enforced"`
}

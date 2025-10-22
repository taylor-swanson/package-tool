package validations

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"package-tool/internal/analyze"
	"package-tool/internal/pkg"
)

const Name = "validations"

var excludeCheckDesc = map[string]string{
	"JSE00001": "Rename message to event.original",
	"PSR00001": "Package with GA version is using an unreleased version of the spec",
	"PSR00002": "Package with non-stable semantic version and active beta features can't be released as stable version",
	"SVR00001": "Saved query, but not filter",
	"SVR00002": "Mandatory filters in dashboard",
	"SVR00003": "Dangling object IDs",
	"SVR00004": "References in dashboard",
	"SVR00005": "Kibana version for saved tags",
	"SVR00006": "Processor tag missgin",
	"SVR00007": "Processor tag duplicated",
}

func fmtExcludeCheckDesc(id string) string {
	if desc, ok := excludeCheckDesc[id]; ok {
		return fmt.Sprintf("%s: %s", id, desc)
	}

	return id
}

var Analyzer = &analyze.Analyzer{
	Name:        Name,
	Description: "Find usages of validations in packages",
	Run:         run,
}

func run(ctx *analyze.Context) (analyze.Result, error) {
	var result analyze.Result

	var validation pkg.Validation
	err := pkg.ReadYAML(filepath.Join(ctx.Package.Path(), "validation.yml"), &validation, true)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return result, err
	}

	// Exclude checks
	for _, v := range validation.Errors.ExcludeChecks {
		// TODO: Filter for specific exclude?
		result.Findings = append(result.Findings, analyze.Finding{
			Category: Name,
			Message:  fmtExcludeCheckDesc(v),
		})
	}

	// Docs structure
	if !validation.DocsStructureEnforced.Enabled {
		result.Findings = append(result.Findings, analyze.Finding{
			Category: Name,
			Message:  "Docs structure is not enforced",
		})
	} else {
		// TODO: Docs structure filters?
	}

	return result, nil
}

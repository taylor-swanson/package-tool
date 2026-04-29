// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package analyze

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fatih/color"
)

func PrintJSON(w io.Writer, report *Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	return enc.Encode(report)
}

func PrintText(w io.Writer, report *Report, wantColor bool) error {
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)
	bold := color.New(color.Bold)

	if !wantColor {
		red.DisableColor()
		yellow.DisableColor()
		green.DisableColor()
		bold.DisableColor()
	}

	var totalIssues int

	var err error
	if _, err = bold.Fprint(w, "Time:"); err != nil {
		return err
	}
	if _, err = fmt.Fprintf(w, " %s\n\n", report.Timestamp.Format(time.RFC3339)); err != nil {
		return err
	}
	if _, err = bold.Fprintln(w, "Analyzers:"); err != nil {
		return err
	}
	for _, name := range report.Analyzers {
		if _, err = fmt.Fprintf(w, "  - %s\n", name); err != nil {
			return err
		}
	}
	if _, err = fmt.Fprintln(w, ""); err != nil {
		return err
	}
	if _, err = bold.Fprintln(w, "Packages:"); err != nil {
		return err
	}
	for _, name := range report.Packages {
		if _, err = fmt.Fprintf(w, "  - %s\n", name); err != nil {
			return err
		}
	}
	if _, err = fmt.Fprintln(w, ""); err != nil {
		return err
	}
	if _, err = bold.Fprintln(w, "Reports:"); err != nil {
		return err
	}
	for _, pkgName := range report.Packages {
		if _, err = bold.Fprintf(w, "-- %s %s\n\n", pkgName, strings.Repeat("-", 76-len(pkgName))); err != nil {
			return err
		}

		// Per-analyzer issues

		for _, analyzerName := range report.Analyzers {
			r := report.Reports[pkgName].Analyzers[analyzerName]
			if len(r.Issues) == 0 {
				continue
			}

			if _, err = bold.Fprintln(w, analyzerName); err != nil {
				return err
			}

			for _, issue := range r.Issues {
				if _, err = fmt.Fprint(w, "  - "); err != nil {
					return err
				}

				switch issue.Severity {
				case SeverityHint:
					if _, err = fmt.Fprint(w, "HINT: "+issue.Message); err != nil {
						return err
					}
				case SeverityInfo:
					if _, err = fmt.Fprint(w, "INFO: "+issue.Message); err != nil {
						return err
					}
				case SeverityWarn:
					if _, err = yellow.Fprint(w, "WARN: "+issue.Message); err != nil {
						return err
					}
				case SeverityError:
					if _, err = red.Fprint(w, "ERROR: "+issue.Message); err != nil {
						return err
					}
				case SeverityDeprecated:
					if _, err = yellow.Fprint(w, "DEPRECATED: "+issue.Message); err != nil {
						return err
					}
				}

				if _, err = fmt.Fprintf(w, " (%s)\n", issue.Analyzer); err != nil {
					return err
				}

				if issue.Pos.IsValid() {
					if _, err = bold.Fprintf(w, "    %s\n", issue.Pos); err != nil {
						return err
					}
				}
			}

			if _, err = fmt.Fprintln(w, ""); err != nil {
				return err
			}
		}

		// Issue totals

		var pkgTotal int
		for _, analyzerName := range report.Analyzers {
			pkgTotal += len(report.Reports[pkgName].Analyzers[analyzerName].Issues)
		}
		totalIssues += pkgTotal
		if pkgTotal > 0 {
			if _, err = red.Fprintf(w, "%d issues:\n", pkgTotal); err != nil {
				return err
			}
		} else {
			if _, err = green.Fprintln(w, "No issues:"); err != nil {
				return err
			}
		}
		for _, analyzerName := range report.Analyzers {
			if _, err = fmt.Fprintf(w, "  - %s: %d\n", analyzerName, len(report.Reports[pkgName].Analyzers[analyzerName].Issues)); err != nil {
				return err
			}
		}
		if _, err = fmt.Fprintln(w, ""); err != nil {
			return err
		}
	}

	return nil
}

// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package analyze

import "encoding/json"

// Severity defines a severity level for analyzer issues.
type Severity int

const (
	// SeverityInfo is an informational message from an analyzer.
	SeverityInfo Severity = iota
	// SeverityHint is a hint or suggestion from an analyzer.
	SeverityHint
	// SeverityWarn is a warning from an analyzer.
	SeverityWarn
	// SeverityError is an error from an analyzer.
	SeverityError
	// SeverityDeprecated is a deprecation notice from an analyzer.
	SeverityDeprecated
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityHint:
		return "hint"
	case SeverityWarn:
		return "warn"
	case SeverityError:
		return "error"
	case SeverityDeprecated:
		return "deprecated"
	}

	return ""
}

func (s Severity) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

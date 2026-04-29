// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package analyze

import "time"

type IssueReport struct {
	Issues []Issue `json:"issues,omitempty"`
}

type PackageReport struct {
	Analyzers map[string]IssueReport `json:"analyzers,omitempty"`
}

type Report struct {
	Timestamp time.Time                `json:"timestamp"`
	Analyzers []string                 `json:"analyzers,omitempty"`
	Packages  []string                 `json:"packages,omitempty"`
	Reports   map[string]PackageReport `json:"reports,omitempty"`
}

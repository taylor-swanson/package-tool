// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package analyze

import "go/token"

type Issue struct {
	Analyzer string         `json:"analyzer"`
	Message  string         `json:"message"`
	Severity Severity       `json:"severity"`
	Pos      token.Position `json:"pos"`
	Related  []Related      `json:"related,omitempty"`
}

type Related struct {
	Pos     token.Position `json:"pos"`
	Message string         `json:"message"`
}

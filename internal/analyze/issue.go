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

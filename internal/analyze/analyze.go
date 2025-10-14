package analyze

import (
	"encoding/json"
	"strconv"

	"github.com/andrewkroh/go-fleetpkg"
)

type Analyzer struct {
	Name        string
	Description string
	CanFix      bool
	Run         func(ctx *Context) (Result, error)
}

type Context struct {
	Package *fleetpkg.Integration
	Fix     bool
}

type Result struct {
	Issues []Issue
	Fixes  []Fix
}

type Issue struct {
	Pos      Pos       `json:"pos"`
	Category string    `json:"category"`
	Message  string    `json:"message"`
	Related  []Related `json:"related,omitempty"`
}

type Fix struct {
	Category string `json:"category"`
	Message  string `json:"message"`
}

type Related struct {
	Pos     Pos    `json:"pos"`
	Message string `json:"message"`
}

type Pos struct {
	File string
	Line int
	Col  int
}

func (p Pos) String() string {
	if p.Line == 0 {
		return p.File
	}
	if p.Col == 0 {
		return p.File + ":" + strconv.Itoa(p.Line)
	}

	return p.File + ":" + strconv.Itoa(p.Line) + ":" + strconv.Itoa(p.Col)
}

func (p Pos) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

func NewPos(meta fleetpkg.FileMetadata) Pos {
	return Pos{
		File: meta.Path(),
		Line: meta.Line(),
		Col:  meta.Column(),
	}
}

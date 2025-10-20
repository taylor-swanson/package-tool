package analyze

import (
	"testing"

	"github.com/goccy/go-yaml/parser"
	"github.com/stretchr/testify/require"
)

func TestNewPosFromPath(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
		path     string
		want     Pos
	}{
		{
			name:     "basic",
			filename: "testdata/basic.yml",
			path:     "$.processors[0]",
			want: Pos{
				File:   "testdata/basic.yml",
				Path:   "$.processors[0]",
				Line:   2,
				Column: 8,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := parser.ParseFile(tc.filename, parser.ParseComments)
			require.NoError(t, err)

			got, err := NewPosFromPath(f, tc.path)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

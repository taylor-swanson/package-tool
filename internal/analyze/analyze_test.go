// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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

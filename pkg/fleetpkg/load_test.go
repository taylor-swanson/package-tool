// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package fleetpkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	testCases := []struct {
		name    string
		dir     string
		wantErr bool
	}{
		{
			name: "ok",
			dir:  "testdata/packages/fortinet_fortigate",
		},
		{
			name:    "invalid",
			dir:     "testdata/packages/invalid",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, gotErr := Load(tc.dir, LoadModeFull)

			if tc.wantErr {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, gotErr)
				assert.NotNil(t, got)
			}
		})
	}
}

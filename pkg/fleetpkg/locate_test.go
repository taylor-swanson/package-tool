// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package fleetpkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocateRoot(t *testing.T) {
	testCases := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name: "ok",
			path: "testdata/packages/fortinet_fortigate",
		},
		{
			name:    "invalid",
			path:    "testdata/packages/invalid",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, gotErr := LocateRoot(tc.path)

			if tc.wantErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
				assert.Contains(t, got, tc.path)
			}
		})
	}
}

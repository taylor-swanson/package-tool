// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/taylor-swanson/package-tool/internal/version"
)

func newCmdVersion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Report version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("%s version %s [commit %v]\n", version.Name, version.Version, version.Commit)
		},
		SilenceUsage:      true,
		SilenceErrors:     true,
		DisableAutoGenTag: true,
	}

	return cmd
}

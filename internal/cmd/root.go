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

package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

func Execute() error {
	cmd := cobra.Command{
		Use:   "package-tool",
		Short: "Tools for Elastic integration packages",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			level := slog.LevelInfo

			logDebug, _ := cmd.Flags().GetBool("debug")
			if logDebug {
				level = slog.LevelDebug
			}

			logJSON, _ := cmd.Flags().GetBool("json")
			if logJSON {
				slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
			} else {
				slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceErrors:     true,
		DisableAutoGenTag: true,
	}

	cmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	cmd.PersistentFlags().Bool("json", false, "enable JSON logging")

	cmd.PersistentFlags().Bool("deprecated", false, "include deprecated packages")
	cmd.PersistentFlags().String("filter-owner", "", "filter by owner of the package")
	cmd.PersistentFlags().String("filter-owner-type", "", "filter by owner type of the package")
	cmd.PersistentFlags().String("filter-type", "", "filter by type of the package")

	cmd.AddCommand(
		newCmdAnalyze(),
		newCmdList(),
		newCmdVersion(),
	)

	return cmd.Execute()
}

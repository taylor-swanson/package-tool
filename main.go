// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package main

import (
	"log/slog"
	"os"

	"github.com/taylor-swanson/package-tool/internal/cmd"
)

func Main() int {
	if err := cmd.Execute(); err != nil {
		slog.Error("Error running command", slog.String("error", err.Error()))
		return 1
	}

	return 0
}

func main() {
	os.Exit(Main())
}

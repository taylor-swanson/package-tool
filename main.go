package main

import (
	"log/slog"
	"os"

	"package-tool/internal/cmd"
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

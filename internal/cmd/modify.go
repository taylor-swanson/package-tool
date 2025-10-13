package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"text/tabwriter"

	"github.com/andrewkroh/go-fleetpkg"
	"github.com/spf13/cobra"

	"package-tool/internal/modify"
	"package-tool/internal/modify/pipelinetag"
)

var modifiers = []*modify.Modifier{
	pipelinetag.Modifier,
}

func getModifier(name string) (*modify.Modifier, bool) {
	for _, m := range modifiers {
		if m.Name == name {
			return m, true
		}
	}

	return nil, false
}

func newCmdModify() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "modify MODIFIER [DIR ...]",
		Short: "Modify package contents",
		RunE:  doModify,
	}

	return cmd
}

func printModifiers() {
	_, _ = fmt.Fprintln(os.Stderr, "Please provide one of the following modifiers:")
	_, _ = fmt.Fprintln(os.Stderr, "")
	tw := tabwriter.NewWriter(os.Stderr, 0, 2, 3, ' ', 0)
	for _, m := range modifiers {
		_, _ = fmt.Fprintf(tw, "%s\t%s\n", m.Name, m.Description)
	}
	_ = tw.Flush()
	_, _ = fmt.Fprintln(os.Stderr, "")
}

func doModify(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		printModifiers()
		return errors.New("no modifier specified")
	}

	modifier, ok := getModifier(args[0])
	if !ok {
		printModifiers()
		return fmt.Errorf("invalid modifier: %s", args[0])
	}

	pkgDirs, err := filterPackages(args[1:], cmd.Flags())
	if err != nil {
		return err
	}

	for _, pkgDir := range pkgDirs {
		pkg, err := fleetpkg.Read(pkgDir)
		if err != nil {
			slog.Error("Failed to read package", slog.String("path", pkgDir), slog.String("error", err.Error()))
			continue
		}

		if err = modifier.Run(pkg); err != nil {
			slog.Error("Failed to run modifier", slog.String("package", pkg.Manifest.Name), slog.String("modifier", modifier.Name), slog.String("error", err.Error()))
			continue
		}
	}

	return nil
}

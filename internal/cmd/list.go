package cmd

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"package-tool/internal/pkg"
)

const (
	listFormatPlain             = "plain"
	listFormatMarkdownList      = "md-list"
	listFormatMarkdownChecklist = "md-checklist"
	listFormatComma             = "comma"
	listFormatCommaLines        = "comma-lines"
	listFormatCustom            = "custom"
)

var listFormats = []string{
	listFormatPlain,
	listFormatMarkdownList,
	listFormatMarkdownChecklist,
	listFormatComma,
	listFormatCustom,
}

func newCmdList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List package contents",
		Aliases: []string{"l"},
		RunE:    doList,
	}

	cmd.Flags().String("custom-prefix", "", "Prefix to add for custom output")
	cmd.Flags().String("custom-suffix", "", "Suffix to add for custom output")
	cmd.Flags().Bool("custom-newline", false, "Add new lines for custom output")
	cmd.Flags().StringP("output", "o", listFormatPlain, "Output format (choose from: "+strings.Join(listFormats, ", ")+")")

	return cmd
}

func doList(cmd *cobra.Command, args []string) error {
	pkgDirs, err := filterPackages(args, cmd.Flags())
	if err != nil {
		return err
	}

	var packages []string
	for _, pkgDir := range pkgDirs {
		manifest, err := pkg.ReadManifest(pkgDir)
		if err != nil {
			slog.Error("Failed to read package", slog.String("path", pkgDir), slog.String("error", err.Error()))
			continue
		}

		packages = append(packages, manifest.Name)
	}

	output, _ := cmd.Flags().GetString("output")
	for i, p := range packages {
		switch output {
		case listFormatMarkdownList:
			_, _ = fmt.Println("- " + p)
		case listFormatMarkdownChecklist:
			_, _ = fmt.Println("- [ ] " + p)
		case listFormatComma:
			_, _ = fmt.Print(p)
			if i < len(packages)-1 {
				_, _ = fmt.Print(", ")
			} else {
				_, _ = fmt.Println("")
			}
		case listFormatCommaLines:
			_, _ = fmt.Println(p + ",")
		case listFormatCustom:
			customPrefix, _ := cmd.Flags().GetString("custom-prefix")
			customSuffix, _ := cmd.Flags().GetString("custom-suffix")
			customNewline, _ := cmd.Flags().GetBool("custom-newline")

			_, _ = fmt.Print(customPrefix)
			_, _ = fmt.Print(p)
			_, _ = fmt.Print(customSuffix)
			if customNewline {
				_, _ = fmt.Println()
			}
		case listFormatPlain:
			fallthrough
		default:
			_, _ = fmt.Println(p)
		}
	}

	return nil
}

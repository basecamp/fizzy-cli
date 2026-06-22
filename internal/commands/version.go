package commands

import (
	"fmt"

	"github.com/basecamp/cli/output"
	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Print version information",
	Example: "$ fizzy version",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfgJQ != "" {
			return errors.ErrJQNotSupported("the version command")
		}
		switch out.EffectiveFormat() {
		case output.FormatStyled, output.FormatMarkdown:
			_, err := fmt.Fprintf(outWriter, "fizzy version %s\n", rootCmd.Version)
			recordOutputError(err)
			captureResponse()
		default:
			printSuccess(map[string]any{
				"version": rootCmd.Version,
			})
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

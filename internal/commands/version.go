package commands

import (
	"fmt"

	"github.com/basecamp/cli/output"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Print version information",
	Example: "$ fizzy version",
	RunE: func(cmd *cobra.Command, args []string) error {
		switch out.EffectiveFormat() {
		case output.FormatStyled, output.FormatMarkdown:
			fmt.Fprintf(outWriter, "fizzy version %s\n", rootCmd.Version)
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

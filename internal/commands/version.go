package commands

import (
	"fmt"

	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfgJQ != "" {
			return errors.ErrJQNotSupported("the version command")
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "fizzy version %s\n", rootCmd.Version)
		return err
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

package commands

import "github.com/spf13/cobra"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		printSuccess(map[string]any{
			"version": rootCmd.Version,
		})
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

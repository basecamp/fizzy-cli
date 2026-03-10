package commands

import (
	"os"

	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload files",
	Long:  "Commands for uploading files for use in rich text fields.",
}

var uploadFileCmd = &cobra.Command{
	Use:   "file PATH",
	Short: "Upload a file",
	Long:  "Uploads a file and returns a signed_id for use in rich text fields.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Upload uses legacy client only — skip SDK initialization
		if err := requireAuth(); err != nil {
			return err
		}
		if err := requireAccount(); err != nil {
			return err
		}

		filePath := args[0]

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return errors.NewError("File not found: " + filePath)
		}

		// UploadFile not available in SDK — keep old client
		client := getClient()
		resp, err := client.UploadFile(filePath)
		if err != nil {
			return err
		}

		printMutation(resp.Data, "", nil)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.AddCommand(uploadFileCmd)
}

package commands

import (
	"fmt"
	"os"

	"github.com/basecamp/fizzy-cli/internal/response"
	"github.com/spf13/cobra"
)

var stepCmd = &cobra.Command{
	Use:   "step",
	Short: "Manage steps (to-do items)",
	Long:  "Commands for managing card steps (to-do items).",
}

// Step show flags
var stepShowCard string

var stepShowCmd = &cobra.Command{
	Use:   "show STEP_ID",
	Short: "Show a step",
	Long:  "Shows details of a specific step.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		if stepShowCard == "" {
			exitWithError(newRequiredFlagError("card"))
		}

		stepID := args[0]
		cardNumber := stepShowCard

		client := getClient()
		resp, err := client.Get("/cards/" + cardNumber + "/steps/" + stepID + ".json")
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("update", fmt.Sprintf("fizzy step update %s --card %s", stepID, cardNumber), "Update step"),
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
		}

		printSuccessWithBreadcrumbs(resp.Data, "", breadcrumbs)
	},
}

// Step create flags
var stepCreateCard string
var stepCreateContent string
var stepCreateCompleted bool

var stepCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a step",
	Long:  "Creates a new step (to-do item) on a card.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		if stepCreateCard == "" {
			exitWithError(newRequiredFlagError("card"))
		}
		if stepCreateContent == "" {
			exitWithError(newRequiredFlagError("content"))
		}

		stepParams := map[string]interface{}{
			"content": stepCreateContent,
		}
		if stepCreateCompleted {
			stepParams["completed"] = true
		}

		body := map[string]interface{}{
			"step": stepParams,
		}

		cardNumber := stepCreateCard

		client := getClient()
		resp, err := client.Post("/cards/"+cardNumber+"/steps.json", body)
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("step", fmt.Sprintf("fizzy step create --card %s --content \"text\"", cardNumber), "Add another step"),
		}

		// Create returns location header - follow it to get the created resource
		if resp.Location != "" {
			followResp, err := client.FollowLocation(resp.Location)
			if err == nil && followResp != nil {
				respObj := response.SuccessWithBreadcrumbs(followResp.Data, "", breadcrumbs)
				respObj.Location = resp.Location
				if lastResult != nil {
					lastResult.Response = respObj
					lastResult.ExitCode = 0
					panic(testExitSignal{})
				}
				respObj.Print()
				os.Exit(0)
				return
			}
			printSuccessWithLocation(resp.Location)
			return
		}

		printSuccessWithBreadcrumbs(resp.Data, "", breadcrumbs)
	},
}

// Step update flags
var stepUpdateCard string
var stepUpdateContent string
var stepUpdateCompleted bool
var stepUpdateNotCompleted bool

var stepUpdateCmd = &cobra.Command{
	Use:   "update STEP_ID",
	Short: "Update a step",
	Long:  "Updates an existing step.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		if stepUpdateCard == "" {
			exitWithError(newRequiredFlagError("card"))
		}

		stepParams := make(map[string]interface{})

		if stepUpdateContent != "" {
			stepParams["content"] = stepUpdateContent
		}
		if stepUpdateCompleted {
			stepParams["completed"] = true
		}
		if stepUpdateNotCompleted {
			stepParams["completed"] = false
		}

		body := map[string]interface{}{
			"step": stepParams,
		}

		stepID := args[0]
		cardNumber := stepUpdateCard

		client := getClient()
		resp, err := client.Patch("/cards/"+cardNumber+"/steps/"+stepID+".json", body)
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy step show %s --card %s", stepID, cardNumber), "View step"),
			breadcrumb("card", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
		}

		printSuccessWithBreadcrumbs(resp.Data, "", breadcrumbs)
	},
}

// Step delete flags
var stepDeleteCard string

var stepDeleteCmd = &cobra.Command{
	Use:   "delete STEP_ID",
	Short: "Delete a step",
	Long:  "Deletes a step from a card.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		if stepDeleteCard == "" {
			exitWithError(newRequiredFlagError("card"))
		}

		cardNumber := stepDeleteCard

		client := getClient()
		_, err := client.Delete("/cards/" + cardNumber + "/steps/" + args[0] + ".json")
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("step", fmt.Sprintf("fizzy step create --card %s --content \"text\"", cardNumber), "Add step"),
		}

		printSuccessWithBreadcrumbs(map[string]interface{}{
			"deleted": true,
		}, "", breadcrumbs)
	},
}

func init() {
	rootCmd.AddCommand(stepCmd)

	// Show
	stepShowCmd.Flags().StringVar(&stepShowCard, "card", "", "Card number (required)")
	stepCmd.AddCommand(stepShowCmd)

	// Create
	stepCreateCmd.Flags().StringVar(&stepCreateCard, "card", "", "Card number (required)")
	stepCreateCmd.Flags().StringVar(&stepCreateContent, "content", "", "Step content (required)")
	stepCreateCmd.Flags().BoolVar(&stepCreateCompleted, "completed", false, "Mark as completed")
	stepCmd.AddCommand(stepCreateCmd)

	// Update
	stepUpdateCmd.Flags().StringVar(&stepUpdateCard, "card", "", "Card number (required)")
	stepUpdateCmd.Flags().StringVar(&stepUpdateContent, "content", "", "Step content")
	stepUpdateCmd.Flags().BoolVar(&stepUpdateCompleted, "completed", false, "Mark as completed")
	stepUpdateCmd.Flags().BoolVar(&stepUpdateNotCompleted, "not_completed", false, "Mark as not completed")
	stepCmd.AddCommand(stepUpdateCmd)

	// Delete
	stepDeleteCmd.Flags().StringVar(&stepDeleteCard, "card", "", "Card number (required)")
	stepCmd.AddCommand(stepDeleteCmd)
}

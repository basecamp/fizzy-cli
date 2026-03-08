package commands

import (
	"fmt"

	"github.com/basecamp/fizzy-sdk/go/pkg/generated"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if stepShowCard == "" {
			return newRequiredFlagError("card")
		}

		stepID := args[0]
		cardNumber := stepShowCard

		ac := getSDK()
		data, _, err := ac.Steps().Get(cmd.Context(), cardNumber, stepID)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("update", fmt.Sprintf("fizzy step update %s --card %s", stepID, cardNumber), "Update step"),
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
		}

		printDetail(normalizeAny(data), "", breadcrumbs)
		return nil
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
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if stepCreateCard == "" {
			return newRequiredFlagError("card")
		}
		if stepCreateContent == "" {
			return newRequiredFlagError("content")
		}

		cardNumber := stepCreateCard
		ac := getSDK()

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("step", fmt.Sprintf("fizzy step create --card %s --content \"text\"", cardNumber), "Add another step"),
		}

		req := &generated.CreateStepRequest{Content: stepCreateContent}
		if stepCreateCompleted {
			req.Completed = true
		}
		data, resp, err := ac.Steps().Create(cmd.Context(), cardNumber, req)
		if err != nil {
			return convertSDKError(err)
		}

		items := normalizeAny(data)
		if location := resp.Headers.Get("Location"); location != "" {
			printMutationWithLocation(items, location, breadcrumbs)
		} else {
			printMutation(items, "", breadcrumbs)
		}
		return nil
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
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if stepUpdateCard == "" {
			return newRequiredFlagError("card")
		}

		stepID := args[0]
		cardNumber := stepUpdateCard

		req := &generated.UpdateStepRequest{}
		if stepUpdateContent != "" {
			req.Content = stepUpdateContent
		}
		if stepUpdateCompleted {
			req.Completed = true
		}
		if stepUpdateNotCompleted {
			req.Completed = false
		}

		ac := getSDK()
		data, _, err := ac.Steps().Update(cmd.Context(), cardNumber, stepID, req)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy step show %s --card %s", stepID, cardNumber), "View step"),
			breadcrumb("card", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
		}

		result := normalizeAny(data)
		if result == nil {
			result = map[string]any{}
		}
		printMutation(result, "", breadcrumbs)
		return nil
	},
}

// Step delete flags
var stepDeleteCard string

var stepDeleteCmd = &cobra.Command{
	Use:   "delete STEP_ID",
	Short: "Delete a step",
	Long:  "Deletes a step from a card.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if stepDeleteCard == "" {
			return newRequiredFlagError("card")
		}

		cardNumber := stepDeleteCard

		_, err := getSDK().Steps().Delete(cmd.Context(), cardNumber, args[0])
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("step", fmt.Sprintf("fizzy step create --card %s --content \"text\"", cardNumber), "Add step"),
		}

		printMutation(map[string]any{
			"deleted": true,
		}, "", breadcrumbs)
		return nil
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

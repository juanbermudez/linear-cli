package cmd

import (
	"context"
	"fmt"
	"sort"

	"github.com/juanbermudez/agent-linear-cli/internal/api"
	"github.com/juanbermudez/agent-linear-cli/internal/cache"
	"github.com/juanbermudez/agent-linear-cli/internal/display"
	"github.com/juanbermudez/agent-linear-cli/internal/output"
	"github.com/spf13/cobra"
)

// LabelResponse represents a label in responses
type LabelResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Color       string  `json:"color"`
	Description string  `json:"description,omitempty"`
	ParentID    *string `json:"parentId,omitempty"`
	TeamID      string  `json:"teamId,omitempty"`
}

// LabelsListResponse is the response for label list
type LabelsListResponse struct {
	Labels []LabelResponse `json:"labels"`
	Count  int             `json:"count"`
}

// NewLabelCmd creates the label command group
func NewLabelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "label",
		Aliases: []string{"l"},
		Short:   "Manage Linear labels",
		Long: `Create, list, update, and delete labels for issues.

Labels are team-scoped and can be organized hierarchically with parent labels.

Examples:
  linear label list --team ENG
  linear label create --name "bug" --color "#FF0000" --team ENG`,
	}

	cmd.AddCommand(newLabelListCmd())
	cmd.AddCommand(newLabelCreateCmd())
	cmd.AddCommand(newLabelUpdateCmd())
	cmd.AddCommand(newLabelDeleteCmd())

	return cmd
}

func newLabelListCmd() *cobra.Command {
	var (
		teamKey string
		plain   bool
		refresh bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List labels for a team",
		Long: `List all labels for a team.

Labels are sorted alphabetically by name.
Results are cached for 24 hours.

Examples:
  linear label list --team ENG
  linear label list --team ENG --plain
  linear label list --team ENG --refresh`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if teamKey == "" {
				teamKey = GetTeamID()
			}
			if teamKey == "" {
				if IsHumanOutput() {
					output.ErrorHuman("Team is required. Use --team flag or configure default team.")
					return nil
				}
				return output.Error("MISSING_TEAM", "Team is required. Use --team flag or configure default team.")
			}

			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			// Resolve team key to ID
			team, err := client.GetTeamByKey(ctx, teamKey)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}
			if team == nil {
				if IsHumanOutput() {
					output.ErrorHuman(fmt.Sprintf("Team '%s' not found", teamKey))
					return nil
				}
				return output.Error("NOT_FOUND", fmt.Sprintf("Team '%s' not found", teamKey))
			}

			var labels *api.LabelsResponse

			// Try cache first
			cacheManager, _ := cache.NewManager()
			cacheKey := cache.TeamKey("labels", team.ID)

			if !refresh && cacheManager != nil {
				cached, _ := cache.Read[api.LabelsResponse](cacheManager, cacheKey)
				if cached != nil {
					labels = cached
				}
			}

			// Fetch if not cached
			if labels == nil {
				labels, err = client.GetLabels(ctx, team.ID)
				if err != nil {
					if IsHumanOutput() {
						output.ErrorHuman(err.Error())
						return nil
					}
					return output.Error("API_ERROR", err.Error())
				}

				// Cache the results
				if cacheManager != nil {
					cache.Write(cacheManager, cacheKey, *labels)
				}
			}

			// Sort alphabetically
			sort.Slice(labels.Labels, func(i, j int) bool {
				return labels.Labels[i].Name < labels.Labels[j].Name
			})

			// Convert to response format
			response := &LabelsListResponse{
				Labels: make([]LabelResponse, len(labels.Labels)),
				Count:  len(labels.Labels),
			}
			for i, l := range labels.Labels {
				response.Labels[i] = LabelResponse{
					ID:    l.ID,
					Name:  l.Name,
					Color: l.Color,
				}
				if l.ParentID != "" {
					response.Labels[i].ParentID = &l.ParentID
				}
			}

			if IsHumanOutput() {
				printLabelsHuman(response, team.Key, plain)
			} else {
				output.JSON(response)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&teamKey, "team", "t", "", "Team key (e.g., ENG)")
	cmd.Flags().BoolVar(&plain, "plain", false, "Plain output without colors")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Bypass cache and fetch fresh data")

	return cmd
}

func newLabelCreateCmd() *cobra.Command {
	var (
		name        string
		description string
		color       string
		teamKey     string
		parentID    string
		isGroup     bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new label",
		Long: `Create a new label for a team.

Color should be in hex format (e.g., #FF0000).
Use --parent to create a child label under an existing label.
Use --is-group to create a label group (parent label).

Examples:
  linear label create --name "bug" --color "#FF0000" --team ENG
  linear label create --name "critical" --parent "bug-label-id" --team ENG
  linear label create --name "Priority" --is-group --team ENG`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				if IsHumanOutput() {
					output.ErrorHuman("Label name is required. Use --name flag.")
					return nil
				}
				return output.Error("MISSING_NAME", "Label name is required. Use --name flag.")
			}

			if teamKey == "" {
				teamKey = GetTeamID()
			}
			if teamKey == "" {
				if IsHumanOutput() {
					output.ErrorHuman("Team is required. Use --team flag or configure default team.")
					return nil
				}
				return output.Error("MISSING_TEAM", "Team is required. Use --team flag or configure default team.")
			}

			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			// Resolve team key to ID
			team, err := client.GetTeamByKey(ctx, teamKey)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}
			if team == nil {
				if IsHumanOutput() {
					output.ErrorHuman(fmt.Sprintf("Team '%s' not found", teamKey))
					return nil
				}
				return output.Error("NOT_FOUND", fmt.Sprintf("Team '%s' not found", teamKey))
			}

			// Create label via GraphQL
			label, err := createLabel(ctx, client, team.ID, name, description, color, parentID, isGroup)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			// Clear cache
			cacheManager, _ := cache.NewManager()
			if cacheManager != nil {
				cacheKey := cache.TeamKey("labels", team.ID)
				cacheManager.Clear(cacheKey)
			}

			response := map[string]interface{}{
				"success":   true,
				"operation": "create",
				"label":     label,
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Created label '%s'", label.Name))
			} else {
				output.JSON(response)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Label name (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Label description")
	cmd.Flags().StringVarP(&color, "color", "c", "", "Label color in hex format (e.g., #FF0000)")
	cmd.Flags().StringVarP(&teamKey, "team", "t", "", "Team key (e.g., ENG)")
	cmd.Flags().StringVarP(&parentID, "parent", "p", "", "Parent label ID for hierarchical labels")
	cmd.Flags().BoolVar(&isGroup, "is-group", false, "Create as a label group (parent label)")

	return cmd
}

func newLabelUpdateCmd() *cobra.Command {
	var (
		name        string
		description string
		color       string
		parentID    string
	)

	cmd := &cobra.Command{
		Use:   "update <label-id>",
		Short: "Update a label",
		Long: `Update an existing label.

At least one field must be provided to update.

Examples:
  linear label update abc123 --name "critical bug"
  linear label update abc123 --color "#00FF00"
  linear label update abc123 --description "Critical issues"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			labelID := args[0]

			// Check that at least one field is provided
			if name == "" && description == "" && color == "" && parentID == "" {
				if IsHumanOutput() {
					output.ErrorHuman("At least one field must be provided to update (--name, --description, --color, --parent)")
					return nil
				}
				return output.Error("MISSING_FIELD", "At least one field must be provided to update")
			}

			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			// Update label via GraphQL
			label, err := updateLabel(ctx, client, labelID, name, description, color, parentID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			response := map[string]interface{}{
				"success":   true,
				"operation": "update",
				"label":     label,
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Updated label '%s'", label.Name))
			} else {
				output.JSON(response)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "New label name")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New label description")
	cmd.Flags().StringVarP(&color, "color", "c", "", "New label color in hex format")
	cmd.Flags().StringVarP(&parentID, "parent", "p", "", "New parent label ID")

	return cmd
}

func newLabelDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <label-id>",
		Aliases: []string{"archive"},
		Short:   "Delete (archive) a label",
		Long: `Delete (archive) a label.

This archives the label, making it unavailable for new issues.
Existing issues with this label will retain it.

Examples:
  linear label delete abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			labelID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			// Delete label via GraphQL
			err = deleteLabel(ctx, client, labelID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman("Label deleted")
			} else {
				output.JSON(map[string]interface{}{
					"success":   true,
					"operation": "delete",
					"labelId":   labelID,
				})
			}

			return nil
		},
	}

	return cmd
}

// createLabel creates a new label via GraphQL
func createLabel(ctx context.Context, client *api.Client, teamID, name, description, color, parentID string, isGroup bool) (*LabelResponse, error) {
	var mutation struct {
		IssueLabelCreate struct {
			Success bool `graphql:"success"`
			Label   struct {
				ID    string `graphql:"id"`
				Name  string `graphql:"name"`
				Color string `graphql:"color"`
			} `graphql:"issueLabel"`
		} `graphql:"issueLabelCreate(input: $input)"`
	}

	input := map[string]interface{}{
		"name":   name,
		"teamId": teamID,
	}
	if description != "" {
		input["description"] = description
	}
	if color != "" {
		input["color"] = color
	}
	if parentID != "" {
		input["parentId"] = parentID
	}
	if isGroup {
		input["isGroup"] = true
	}

	variables := map[string]interface{}{
		"input": input,
	}

	if err := client.Mutate(ctx, &mutation, variables); err != nil {
		return nil, err
	}

	if !mutation.IssueLabelCreate.Success {
		return nil, fmt.Errorf("failed to create label")
	}

	return &LabelResponse{
		ID:    mutation.IssueLabelCreate.Label.ID,
		Name:  mutation.IssueLabelCreate.Label.Name,
		Color: mutation.IssueLabelCreate.Label.Color,
	}, nil
}

// updateLabel updates a label via GraphQL
func updateLabel(ctx context.Context, client *api.Client, labelID, name, description, color, parentID string) (*LabelResponse, error) {
	var mutation struct {
		IssueLabelUpdate struct {
			Success bool `graphql:"success"`
			Label   struct {
				ID    string `graphql:"id"`
				Name  string `graphql:"name"`
				Color string `graphql:"color"`
			} `graphql:"issueLabel"`
		} `graphql:"issueLabelUpdate(id: $id, input: $input)"`
	}

	input := map[string]interface{}{}
	if name != "" {
		input["name"] = name
	}
	if description != "" {
		input["description"] = description
	}
	if color != "" {
		input["color"] = color
	}
	if parentID != "" {
		input["parentId"] = parentID
	}

	variables := map[string]interface{}{
		"id":    labelID,
		"input": input,
	}

	if err := client.Mutate(ctx, &mutation, variables); err != nil {
		return nil, err
	}

	if !mutation.IssueLabelUpdate.Success {
		return nil, fmt.Errorf("failed to update label")
	}

	return &LabelResponse{
		ID:    mutation.IssueLabelUpdate.Label.ID,
		Name:  mutation.IssueLabelUpdate.Label.Name,
		Color: mutation.IssueLabelUpdate.Label.Color,
	}, nil
}

// deleteLabel archives a label via GraphQL
func deleteLabel(ctx context.Context, client *api.Client, labelID string) error {
	var mutation struct {
		IssueLabelArchive struct {
			Success bool `graphql:"success"`
		} `graphql:"issueLabelArchive(id: $id)"`
	}

	variables := map[string]interface{}{
		"id": labelID,
	}

	if err := client.Mutate(ctx, &mutation, variables); err != nil {
		return err
	}

	if !mutation.IssueLabelArchive.Success {
		return fmt.Errorf("failed to delete label")
	}

	return nil
}

func printLabelsHuman(labels *LabelsListResponse, teamKey string, plain bool) {
	if len(labels.Labels) == 0 {
		output.HumanLn("No labels found for team %s", teamKey)
		return
	}

	output.HumanLn("Labels for team %s:\n", teamKey)

	headers := []string{"NAME", "COLOR", "ID"}
	rows := make([][]string, len(labels.Labels))

	for i, l := range labels.Labels {
		colorDisplay := l.Color
		if !plain {
			colorDisplay = display.ColorBox(l.Color) + " " + l.Color
		}

		rows[i] = []string{
			l.Name,
			colorDisplay,
			output.Muted("%s", l.ID),
		}
	}

	output.TableWithColors(headers, rows)
	output.HumanLn("\n%d labels", labels.Count)
}

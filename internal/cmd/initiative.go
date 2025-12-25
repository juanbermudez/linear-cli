package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/juanbermudez/agent-linear-cli/internal/api"
	"github.com/juanbermudez/agent-linear-cli/internal/display"
	"github.com/juanbermudez/agent-linear-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewInitiativeCmd creates the initiative command group
func NewInitiativeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "initiative",
		Aliases: []string{"init", "i"},
		Short:   "Manage Linear initiatives",
		Long: `Create, view, update, and manage Linear initiatives.

Examples:
  linear initiative list
  linear initiative view <initiative-id>
  linear initiative create --name "Q1 Goals"`,
	}

	cmd.AddCommand(newInitiativeListCmd())
	cmd.AddCommand(newInitiativeViewCmd())
	cmd.AddCommand(newInitiativeCreateCmd())
	cmd.AddCommand(newInitiativeUpdateCmd())
	cmd.AddCommand(newInitiativeArchiveCmd())
	cmd.AddCommand(newInitiativeRestoreCmd())
	cmd.AddCommand(newInitiativeProjectAddCmd())
	cmd.AddCommand(newInitiativeProjectRemoveCmd())

	return cmd
}

func newInitiativeListCmd() *cobra.Command {
	var (
		status  string
		ownerID string
		limit   int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List initiatives",
		Long: `List initiatives with optional filters.

Status values: Planned, Active, Completed

Examples:
  linear initiative list
  linear initiative list --status Active
  linear initiative list --limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			initiatives, err := client.GetInitiatives(ctx, status, ownerID, limit)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				printInitiativesHuman(initiatives)
			} else {
				output.JSON(initiatives)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status (Planned, Active, Completed)")
	cmd.Flags().StringVarP(&ownerID, "owner", "o", "", "Filter by owner ID")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Maximum initiatives to return")

	return cmd
}

func newInitiativeViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view <initiative-id>",
		Short: "View initiative details",
		Long: `View detailed information about an initiative.

Examples:
  linear initiative view abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			initiativeID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			initiative, err := client.GetInitiative(ctx, initiativeID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if initiative == nil {
				if IsHumanOutput() {
					output.ErrorHuman(fmt.Sprintf("Initiative '%s' not found", initiativeID))
					return nil
				}
				return output.Error("NOT_FOUND", fmt.Sprintf("Initiative '%s' not found", initiativeID))
			}

			if IsHumanOutput() {
				printInitiativeDetailHuman(initiative)
			} else {
				output.JSON(initiative)
			}

			return nil
		},
	}

	return cmd
}

func newInitiativeCreateCmd() *cobra.Command {
	var (
		name        string
		description string
		content     string
		status      string
		ownerID     string
		targetDate  string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new initiative",
		Long: `Create a new initiative in Linear.

Status values: Planned, Active, Completed

Examples:
  linear initiative create --name "Q1 Goals"
  linear initiative create --name "Platform Redesign" --status Active
  linear initiative create --name "2025 Roadmap" --target-date 2025-12-31`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				if IsHumanOutput() {
					output.ErrorHuman("Initiative name is required. Use --name flag.")
					return nil
				}
				return output.Error("MISSING_NAME", "Initiative name is required")
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

			input := api.InitiativeCreateInput{
				Name:        name,
				Description: description,
				Content:     content,
				Status:      status,
				OwnerID:     ownerID,
				TargetDate:  targetDate,
			}

			initiative, err := client.CreateInitiative(ctx, input)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Initiative created: %s", initiative.Name))
				output.HumanLn("  ID: %s", initiative.ID)
				output.HumanLn("  Status: %s", initiative.Status)
			} else {
				output.JSON(map[string]interface{}{
					"success":    true,
					"operation":  "create",
					"initiative": initiative,
				})
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Initiative name (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Initiative description")
	cmd.Flags().StringVarP(&content, "content", "c", "", "Initiative content (markdown)")
	cmd.Flags().StringVarP(&status, "status", "s", "", "Initiative status (Planned, Active, Completed)")
	cmd.Flags().StringVarP(&ownerID, "owner", "o", "", "Owner user ID")
	cmd.Flags().StringVarP(&targetDate, "target-date", "t", "", "Target date (YYYY-MM-DD)")

	return cmd
}

func newInitiativeUpdateCmd() *cobra.Command {
	var (
		name        string
		description string
		content     string
		status      string
		ownerID     string
		targetDate  string
	)

	cmd := &cobra.Command{
		Use:   "update <initiative-id>",
		Short: "Update an initiative",
		Long: `Update an existing initiative.

Examples:
  linear initiative update abc123 --name "New Name"
  linear initiative update abc123 --status Completed
  linear initiative update abc123 --target-date 2025-06-30`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			initiativeID := args[0]

			// Check if at least one field is being updated
			if !cmd.Flags().Changed("name") &&
				!cmd.Flags().Changed("description") &&
				!cmd.Flags().Changed("content") &&
				!cmd.Flags().Changed("status") &&
				!cmd.Flags().Changed("owner") &&
				!cmd.Flags().Changed("target-date") {
				if IsHumanOutput() {
					output.ErrorHuman("At least one field must be specified to update")
					return nil
				}
				return output.Error("MISSING_FIELDS", "At least one field must be specified to update")
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

			input := api.InitiativeUpdateInput{}

			if cmd.Flags().Changed("name") {
				input.Name = name
			}
			if cmd.Flags().Changed("description") {
				input.Description = description
			}
			if cmd.Flags().Changed("content") {
				input.Content = content
			}
			if cmd.Flags().Changed("status") {
				input.Status = status
			}
			if cmd.Flags().Changed("owner") {
				input.OwnerID = ownerID
			}
			if cmd.Flags().Changed("target-date") {
				input.TargetDate = targetDate
			}

			initiative, err := client.UpdateInitiative(ctx, initiativeID, input)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Initiative updated: %s", initiative.Name))
			} else {
				output.JSON(map[string]interface{}{
					"success":    true,
					"operation":  "update",
					"initiative": initiative,
				})
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Initiative name")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Initiative description")
	cmd.Flags().StringVarP(&content, "content", "c", "", "Initiative content (markdown)")
	cmd.Flags().StringVarP(&status, "status", "s", "", "Initiative status (Planned, Active, Completed)")
	cmd.Flags().StringVarP(&ownerID, "owner", "o", "", "Owner user ID")
	cmd.Flags().StringVarP(&targetDate, "target-date", "t", "", "Target date (YYYY-MM-DD)")

	return cmd
}

func newInitiativeArchiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive <initiative-id>",
		Short: "Archive an initiative",
		Long: `Archive an initiative.

Examples:
  linear initiative archive abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			initiativeID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			err = client.ArchiveInitiative(ctx, initiativeID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman("Initiative archived")
			} else {
				output.JSON(map[string]interface{}{
					"success":      true,
					"operation":    "archive",
					"initiativeId": initiativeID,
				})
			}

			return nil
		},
	}

	return cmd
}

func newInitiativeRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <initiative-id>",
		Short: "Restore an archived initiative",
		Long: `Restore an archived initiative.

Examples:
  linear initiative restore abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			initiativeID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			err = client.RestoreInitiative(ctx, initiativeID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman("Initiative restored")
			} else {
				output.JSON(map[string]interface{}{
					"success":      true,
					"operation":    "restore",
					"initiativeId": initiativeID,
				})
			}

			return nil
		},
	}

	return cmd
}

func newInitiativeProjectAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project-add <initiative-id> <project-id>",
		Short: "Add a project to an initiative",
		Long: `Add a project to an initiative.

Examples:
  linear initiative project-add abc123 xyz789`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			initiativeID := args[0]
			projectID := args[1]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			err = client.AddProjectToInitiative(ctx, initiativeID, projectID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman("Project added to initiative")
			} else {
				output.JSON(map[string]interface{}{
					"success":      true,
					"operation":    "project-add",
					"initiativeId": initiativeID,
					"projectId":    projectID,
				})
			}

			return nil
		},
	}

	return cmd
}

func newInitiativeProjectRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project-remove <initiative-id> <project-id>",
		Short: "Remove a project from an initiative",
		Long: `Remove a project from an initiative.

Examples:
  linear initiative project-remove abc123 xyz789`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			initiativeID := args[0]
			projectID := args[1]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			err = client.RemoveProjectFromInitiative(ctx, initiativeID, projectID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman("Project removed from initiative")
			} else {
				output.JSON(map[string]interface{}{
					"success":      true,
					"operation":    "project-remove",
					"initiativeId": initiativeID,
					"projectId":    projectID,
				})
			}

			return nil
		},
	}

	return cmd
}

// Human output formatters

func printInitiativesHuman(initiatives *api.InitiativesResponse) {
	if len(initiatives.Initiatives) == 0 {
		output.HumanLn("No initiatives found")
		return
	}

	headers := []string{"NAME", "STATUS", "OWNER", "PROJECTS", "TARGET", "ID"}
	rows := make([][]string, len(initiatives.Initiatives))

	for i, init := range initiatives.Initiatives {
		ownerName := "-"
		if init.Owner != nil {
			ownerName = init.Owner.DisplayName
		}

		targetDate := "-"
		if init.TargetDate != "" {
			if t, err := time.Parse("2006-01-02", init.TargetDate); err == nil {
				targetDate = t.Format("Jan 02, 2006")
			} else {
				targetDate = init.TargetDate
			}
		}

		rows[i] = []string{
			display.Truncate(init.Name, 35),
			init.Status,
			ownerName,
			fmt.Sprintf("%d", init.ProjectCount),
			targetDate,
			output.Muted("%s", init.ID),
		}
	}

	output.TableWithColors(headers, rows)
	output.HumanLn("\n%d initiatives", initiatives.Count)
}

func printInitiativeDetailHuman(init *api.Initiative) {
	output.HumanLn("%s", init.Name)
	output.HumanLn("")

	output.HumanLn("Status: %s", init.Status)

	if init.Owner != nil {
		output.HumanLn("Owner: %s", init.Owner.DisplayName)
	}

	if init.TargetDate != "" {
		targetDate := init.TargetDate
		if t, err := time.Parse("2006-01-02", init.TargetDate); err == nil {
			targetDate = t.Format("Jan 02, 2006")
		}
		output.HumanLn("Target Date: %s", targetDate)
	}

	if init.CreatedAt != "" {
		createdAt := init.CreatedAt
		if t, err := time.Parse(time.RFC3339, init.CreatedAt); err == nil {
			createdAt = display.TimeAgo(t)
		}
		output.HumanLn("Created: %s", createdAt)
	}

	if init.UpdatedAt != "" {
		updatedAt := init.UpdatedAt
		if t, err := time.Parse(time.RFC3339, init.UpdatedAt); err == nil {
			updatedAt = display.TimeAgo(t)
		}
		output.HumanLn("Updated: %s", updatedAt)
	}

	output.HumanLn("")
	output.HumanLn("ID: %s", output.Muted("%s", init.ID))

	if len(init.Projects) > 0 {
		output.HumanLn("")
		output.HumanLn("Projects:")
		for _, p := range init.Projects {
			output.HumanLn("  - %s (%s)", p.Name, output.Muted("%s", p.ID))
		}
	}

	if init.Description != "" {
		output.HumanLn("")
		output.HumanLn("Description:")
		output.HumanLn("%s", init.Description)
	}

	if init.Content != "" {
		output.HumanLn("")
		output.HumanLn("Content:")
		output.HumanLn("%s", init.Content)
	}
}

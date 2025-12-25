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

// NewDocumentCmd creates the document command group
func NewDocumentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "document",
		Aliases: []string{"doc", "d"},
		Short:   "Manage Linear documents",
		Long: `Create, view, update, and manage Linear documents.

Examples:
  linear document list
  linear document view <document-id>
  linear document create --title "PRD: Feature X"`,
	}

	cmd.AddCommand(newDocumentListCmd())
	cmd.AddCommand(newDocumentViewCmd())
	cmd.AddCommand(newDocumentCreateCmd())
	cmd.AddCommand(newDocumentUpdateCmd())
	cmd.AddCommand(newDocumentDeleteCmd())
	cmd.AddCommand(newDocumentRestoreCmd())
	cmd.AddCommand(newDocumentSearchCmd())

	return cmd
}

func newDocumentListCmd() *cobra.Command {
	var (
		projectID string
		limit     int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List documents",
		Long: `List documents with optional filters.

Examples:
  linear document list
  linear document list --project abc123
  linear document list --limit 20`,
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

			documents, err := client.GetDocuments(ctx, projectID, limit)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				printDocumentsHuman(documents)
			} else {
				output.JSON(documents)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Filter by project ID")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Maximum documents to return")

	return cmd
}

func newDocumentViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view <document-id>",
		Short: "View document details",
		Long: `View detailed information about a document.

Examples:
  linear document view abc123
  linear document view abc123 --human`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			documentID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			document, err := client.GetDocument(ctx, documentID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if document == nil {
				if IsHumanOutput() {
					output.ErrorHuman(fmt.Sprintf("Document '%s' not found", documentID))
					return nil
				}
				return output.Error("NOT_FOUND", fmt.Sprintf("Document '%s' not found", documentID))
			}

			if IsHumanOutput() {
				printDocumentDetailHuman(document)
			} else {
				output.JSON(document)
			}

			return nil
		},
	}

	return cmd
}

func newDocumentCreateCmd() *cobra.Command {
	var (
		title     string
		content   string
		projectID string
		teamKey   string
		icon      string
		color     string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new document",
		Long: `Create a new document in Linear.

Note: Documents must be associated with a project or team.

Examples:
  linear document create --title "PRD: Feature X" --team ENG
  linear document create --title "Research Notes" --content "## Summary..." --project abc123
  linear document create --title "Spec" --project abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				if IsHumanOutput() {
					output.ErrorHumanWithHint(
						"Document title is required",
						"Provide a title using the --title flag",
						"linear document create --title \"My Doc\" --team ENG",
					)
					return nil
				}
				return output.ErrorWithHint(
					"MISSING_TITLE",
					"Document title is required",
					"Provide a title using the --title flag",
					"linear document create --title \"My Doc\" --team ENG",
				)
			}

			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHumanWithHint(
						err.Error(),
						"Authentication failed. Make sure you're logged in",
						"linear auth login --with-token",
					)
					return nil
				}
				return output.ErrorWithHint(
					"AUTH_ERROR",
					err.Error(),
					"Authentication failed. Make sure you're logged in",
					"linear auth login --with-token",
				)
			}

			// Resolve team key to ID if provided
			var teamID string
			if teamKey == "" {
				teamKey = GetTeamID()
			}
			if teamKey != "" && projectID == "" {
				team, err := client.GetTeamByKey(ctx, teamKey)
				if err != nil {
					if IsHumanOutput() {
						output.ErrorHuman(err.Error())
						return nil
					}
					return output.Error("API_ERROR", err.Error())
				}
				if team != nil {
					teamID = team.ID
				}
			}

			// Ensure we have at least a project or team
			if projectID == "" && teamID == "" {
				if IsHumanOutput() {
					output.ErrorHumanWithHint(
						"Either --project or --team is required",
						"Documents must be associated with a project or team",
						"linear document create --title \"My Doc\" --team ENG",
						"linear document create --title \"My Doc\" --project <project-id>",
					)
					return nil
				}
				return output.ErrorWithHint(
					"MISSING_ASSOCIATION",
					"Either --project or --team is required",
					"Documents must be associated with a project or team",
					"linear document create --title \"My Doc\" --team ENG",
					"linear document create --title \"My Doc\" --project <project-id>",
				)
			}

			input := api.DocumentCreateInput{
				Title:     title,
				Content:   content,
				ProjectID: projectID,
				TeamID:    teamID,
				Icon:      icon,
				Color:     color,
			}

			document, err := client.CreateDocument(ctx, input)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Document created: %s", document.Title))
				output.HumanLn("  ID: %s", document.ID)
				output.HumanLn("  URL: %s", document.URL)
			} else {
				output.JSON(map[string]interface{}{
					"success":   true,
					"operation": "create",
					"document":  document,
				})
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Document title (required)")
	cmd.Flags().StringVarP(&content, "content", "c", "", "Document content (markdown)")
	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Project ID to attach document to")
	cmd.Flags().StringVar(&teamKey, "team", "", "Team key (e.g., ENG)")
	cmd.Flags().StringVarP(&icon, "icon", "i", "", "Document icon")
	cmd.Flags().StringVar(&color, "color", "", "Document color (#RRGGBB)")

	return cmd
}

func newDocumentUpdateCmd() *cobra.Command {
	var (
		title     string
		content   string
		projectID string
		icon      string
		color     string
	)

	cmd := &cobra.Command{
		Use:   "update <document-id>",
		Short: "Update a document",
		Long: `Update an existing document.

Examples:
  linear document update abc123 --title "New Title"
  linear document update abc123 --content "Updated content..."
  linear document update abc123 --project xyz789`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			documentID := args[0]

			// Check if at least one field is being updated
			if !cmd.Flags().Changed("title") &&
				!cmd.Flags().Changed("content") &&
				!cmd.Flags().Changed("project") &&
				!cmd.Flags().Changed("icon") &&
				!cmd.Flags().Changed("color") {
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

			input := api.DocumentUpdateInput{}

			if cmd.Flags().Changed("title") {
				input.Title = title
			}
			if cmd.Flags().Changed("content") {
				input.Content = content
			}
			if cmd.Flags().Changed("project") {
				input.ProjectID = projectID
			}
			if cmd.Flags().Changed("icon") {
				input.Icon = icon
			}
			if cmd.Flags().Changed("color") {
				input.Color = color
			}

			document, err := client.UpdateDocument(ctx, documentID, input)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Document updated: %s", document.Title))
			} else {
				output.JSON(map[string]interface{}{
					"success":   true,
					"operation": "update",
					"document":  document,
				})
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Document title")
	cmd.Flags().StringVarP(&content, "content", "c", "", "Document content (markdown)")
	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Project ID to attach document to")
	cmd.Flags().StringVarP(&icon, "icon", "i", "", "Document icon")
	cmd.Flags().StringVar(&color, "color", "", "Document color (#RRGGBB)")

	return cmd
}

func newDocumentDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <document-id>",
		Short: "Delete a document",
		Long: `Delete a document.

Examples:
  linear document delete abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			documentID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			err = client.DeleteDocument(ctx, documentID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman("Document deleted")
			} else {
				output.JSON(map[string]interface{}{
					"success":    true,
					"operation":  "delete",
					"documentId": documentID,
				})
			}

			return nil
		},
	}

	return cmd
}

func newDocumentRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <document-id>",
		Short: "Restore a deleted document",
		Long: `Restore a previously deleted document.

Examples:
  linear document restore abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			documentID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			err = client.RestoreDocument(ctx, documentID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman("Document restored")
			} else {
				output.JSON(map[string]interface{}{
					"success":    true,
					"operation":  "restore",
					"documentId": documentID,
				})
			}

			return nil
		},
	}

	return cmd
}

func newDocumentSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search documents",
		Long: `Search for documents by title or content.

Examples:
  linear document search "feature"
  linear document search "PRD" --limit 10`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			results, err := client.SearchDocuments(ctx, query, limit)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				printDocumentSearchHuman(results)
			} else {
				output.JSON(results)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum results to return")

	return cmd
}

// Human output formatters

func printDocumentsHuman(documents *api.DocumentsResponse) {
	if len(documents.Documents) == 0 {
		output.HumanLn("No documents found")
		return
	}

	headers := []string{"TITLE", "PROJECT", "CREATOR", "UPDATED", "ID"}
	rows := make([][]string, len(documents.Documents))

	for i, d := range documents.Documents {
		projectName := "-"
		if d.Project != nil {
			projectName = d.Project.Name
		}

		creatorName := "-"
		if d.Creator != nil {
			creatorName = d.Creator.DisplayName
		}

		updatedAt := d.UpdatedAt
		if t, err := time.Parse(time.RFC3339, d.UpdatedAt); err == nil {
			updatedAt = display.TimeAgo(t)
		}

		rows[i] = []string{
			display.Truncate(d.Title, 40),
			display.Truncate(projectName, 20),
			creatorName,
			updatedAt,
			output.Muted("%s", d.ID),
		}
	}

	output.TableWithColors(headers, rows)
	output.HumanLn("\n%d documents", documents.Count)
}

func printDocumentDetailHuman(d *api.Document) {
	output.HumanLn("%s", d.Title)
	output.HumanLn("")

	if d.Creator != nil {
		output.HumanLn("Creator: %s", d.Creator.DisplayName)
	}

	if d.Project != nil {
		output.HumanLn("Project: %s", d.Project.Name)
	}

	if d.CreatedAt != "" {
		createdAt := d.CreatedAt
		if t, err := time.Parse(time.RFC3339, d.CreatedAt); err == nil {
			createdAt = display.TimeAgo(t)
		}
		output.HumanLn("Created: %s", createdAt)
	}

	if d.UpdatedAt != "" {
		updatedAt := d.UpdatedAt
		if t, err := time.Parse(time.RFC3339, d.UpdatedAt); err == nil {
			updatedAt = display.TimeAgo(t)
		}
		output.HumanLn("Updated: %s", updatedAt)
	}

	output.HumanLn("")
	output.HumanLn("URL: %s", d.URL)
	output.HumanLn("ID: %s", output.Muted("%s", d.ID))

	if d.Content != "" {
		output.HumanLn("")
		output.HumanLn("Content:")
		output.HumanLn("%s", d.Content)
	}
}

func printDocumentSearchHuman(results *api.DocumentSearchResponse) {
	if len(results.Documents) == 0 {
		output.HumanLn("No documents found matching '%s'", results.Query)
		return
	}

	output.HumanLn("Search results for '%s':\n", results.Query)

	headers := []string{"TITLE", "PROJECT", "CREATOR", "UPDATED", "ID"}
	rows := make([][]string, len(results.Documents))

	for i, d := range results.Documents {
		projectName := "-"
		if d.Project != nil {
			projectName = d.Project.Name
		}

		creatorName := "-"
		if d.Creator != nil {
			creatorName = d.Creator.DisplayName
		}

		updatedAt := d.UpdatedAt
		if t, err := time.Parse(time.RFC3339, d.UpdatedAt); err == nil {
			updatedAt = display.TimeAgo(t)
		}

		rows[i] = []string{
			display.Truncate(d.Title, 40),
			display.Truncate(projectName, 20),
			creatorName,
			updatedAt,
			output.Muted("%s", d.ID),
		}
	}

	output.TableWithColors(headers, rows)
	output.HumanLn("\n%d of %d documents", results.Count, results.TotalCount)
}

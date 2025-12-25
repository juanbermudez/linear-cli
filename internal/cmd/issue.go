package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/juanbermudez/agent-linear-cli/internal/api"
	"github.com/juanbermudez/agent-linear-cli/internal/display"
	"github.com/juanbermudez/agent-linear-cli/internal/output"
	"github.com/spf13/cobra"
)

// IssueListResponse is the response for issue list command
type IssueListResponse struct {
	Issues []api.IssueListItem `json:"issues"`
	Count  int                 `json:"count"`
}

// NewIssueCmd creates the issue command group
func NewIssueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "issue",
		Aliases: []string{"i"},
		Short:   "Manage Linear issues",
		Long: `Create, view, update, delete, and search Linear issues.

Examples:
  linear issue list --team ENG
  linear issue view ENG-123
  linear issue create --title "Fix bug" --team ENG
  linear issue search "authentication"`,
	}

	// Add subcommands
	cmd.AddCommand(newIssueListCmd())
	cmd.AddCommand(newIssueViewCmd())
	cmd.AddCommand(newIssueCreateCmd())
	cmd.AddCommand(newIssueUpdateCmd())
	cmd.AddCommand(newIssueDeleteCmd())
	cmd.AddCommand(newIssueSearchCmd())
	cmd.AddCommand(newIssueRelateCmd())
	cmd.AddCommand(newIssueUnrelateCmd())
	cmd.AddCommand(newIssueRelationsCmd())
	cmd.AddCommand(newIssueCommentCmd())
	cmd.AddCommand(newIssueAttachmentCmd())

	// Utility commands
	cmd.AddCommand(newIssueStartCmd())
	cmd.AddCommand(newIssueTitleCmd())
	cmd.AddCommand(newIssueURLCmd())
	cmd.AddCommand(newIssueDescribeCmd())

	return cmd
}

func newIssueListCmd() *cobra.Command {
	var (
		stateTypes    []string
		allStates     bool
		assignee      string
		allAssignees  bool
		unassigned    bool
		sortBy        string
		teamKey       string
		projectID     string
		limit         int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issues",
		Long: `List issues with optional filters.

State types: triage, backlog, unstarted, started, completed, canceled

Examples:
  linear issue list --team ENG
  linear issue list --state started --state unstarted
  linear issue list --all-states
  linear issue list --assignee self
  linear issue list --unassigned
  linear issue list --limit 100`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if teamKey == "" {
				teamKey = GetTeamID()
			}
			if teamKey == "" {
				if IsHumanOutput() {
					output.ErrorHumanWithHint(
						"Team is required",
						"Specify a team using --team flag or set a default team",
						"linear issue list --team ENG",
						"linear config set team_key ENG",
					)
					return nil
				}
				return output.ErrorWithHint(
					"MISSING_TEAM",
					"Team is required",
					"Specify a team using --team flag or set a default team",
					"linear issue list --team ENG",
					"linear config set team_key ENG",
				)
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

			// Build filter
			filter := api.IssueFilter{
				TeamID:    team.ID,
				ProjectID: projectID,
			}

			// Handle state filtering
			if !allStates {
				if len(stateTypes) > 0 {
					filter.StateTypes = stateTypes
				} else {
					// Default: show active issues (not completed/canceled)
					filter.StateTypes = []string{"triage", "backlog", "unstarted", "started"}
				}
			}

			// Handle assignee filtering
			if unassigned {
				filter.Unassigned = true
			} else if !allAssignees && assignee != "" {
				if assignee == "self" || assignee == "me" {
					viewerID, err := client.GetViewerID(ctx)
					if err != nil {
						if IsHumanOutput() {
							output.ErrorHuman("Failed to get current user: " + err.Error())
							return nil
						}
						return output.Error("API_ERROR", "Failed to get current user: "+err.Error())
					}
					filter.AssigneeID = viewerID
				} else {
					filter.AssigneeID = assignee
				}
			}

			issues, err := client.GetIssues(ctx, filter, limit, sortBy)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			response := &IssueListResponse{
				Issues: issues.Issues,
				Count:  issues.Count,
			}

			if IsHumanOutput() {
				printIssuesHuman(response, team.Key)
			} else {
				output.JSON(response)
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVarP(&stateTypes, "state", "s", nil, "Filter by state type (triage, backlog, unstarted, started, completed, canceled)")
	cmd.Flags().BoolVar(&allStates, "all-states", false, "Show all states including completed and canceled")
	cmd.Flags().StringVarP(&assignee, "assignee", "a", "", "Filter by assignee (use 'self' for yourself)")
	cmd.Flags().BoolVarP(&allAssignees, "all-assignees", "A", false, "Show issues from all assignees")
	cmd.Flags().BoolVarP(&unassigned, "unassigned", "U", false, "Show only unassigned issues")
	cmd.Flags().StringVar(&sortBy, "sort", "manual", "Sort order (manual, priority)")
	cmd.Flags().StringVarP(&teamKey, "team", "t", "", "Team key (e.g., ENG)")
	cmd.Flags().StringVar(&projectID, "project", "", "Filter by project ID")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Maximum number of issues to return")

	return cmd
}

func newIssueViewCmd() *cobra.Command {
	var (
		noComments bool
	)

	cmd := &cobra.Command{
		Use:   "view <issue-id>",
		Short: "View issue details",
		Long: `View detailed information about a specific issue.

Issue ID can be an identifier (ENG-123) or UUID.

Examples:
  linear issue view ENG-123
  linear issue view ENG-123 --no-comments`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
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

			issue, err := client.GetIssue(ctx, issueID, !noComments)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHumanWithHint(
						err.Error(),
						"Issue not found or invalid ID. Use format TEAM-123 or UUID",
						"linear issue view ENG-123",
						"linear issue search \"keyword\"",
					)
					return nil
				}
				return output.ErrorWithHint(
					"API_ERROR",
					err.Error(),
					"Issue not found or invalid ID. Use format TEAM-123 or UUID",
					"linear issue view ENG-123",
					"linear issue search \"keyword\"",
				)
			}

			if IsHumanOutput() {
				printIssueDetailHuman(issue)
			} else {
				output.JSON(issue)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&noComments, "no-comments", false, "Exclude comments from output")

	return cmd
}

func newIssueCreateCmd() *cobra.Command {
	var (
		title       string
		description string
		priority    int
		estimate    float64
		assignee    string
		labels      []string
		projectID   string
		stateID     string
		teamKey     string
		parentID    string
		dueDate     string
		cycleID     string
		milestoneID string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new issue",
		Long: `Create a new issue in Linear.

Priority values: 0=none, 1=urgent, 2=high, 3=medium, 4=low

Examples:
  linear issue create --title "Fix login bug" --team ENG
  linear issue create --title "Feature" --description "Details..." --priority 2 --team ENG
  linear issue create --title "Subtask" --parent ENG-123 --team ENG`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				if IsHumanOutput() {
					output.ErrorHumanWithHint(
						"Title is required",
						"Provide a title using the --title flag",
						"linear issue create --title \"Fix bug\" --team ENG",
					)
					return nil
				}
				return output.ErrorWithHint(
					"MISSING_TITLE",
					"Title is required",
					"Provide a title using the --title flag",
					"linear issue create --title \"Fix bug\" --team ENG",
				)
			}

			if teamKey == "" {
				teamKey = GetTeamID()
			}
			if teamKey == "" {
				if IsHumanOutput() {
					output.ErrorHumanWithHint(
						"Team is required",
						"Specify a team using --team flag or set a default team",
						"linear issue create --title \"Fix bug\" --team ENG",
						"linear config set team_key ENG",
					)
					return nil
				}
				return output.ErrorWithHint(
					"MISSING_TEAM",
					"Team is required",
					"Specify a team using --team flag or set a default team",
					"linear issue create --title \"Fix bug\" --team ENG",
					"linear config set team_key ENG",
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
						"linear config setup --api-key lin_api_xxx",
					)
					return nil
				}
				return output.ErrorWithHint(
					"AUTH_ERROR",
					err.Error(),
					"Authentication failed. Make sure you're logged in",
					"linear auth login --with-token",
					"linear config setup --api-key lin_api_xxx",
				)
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
					output.ErrorHumanWithHint(
						fmt.Sprintf("Team '%s' not found", teamKey),
						"Check available teams and use a valid team key",
						"linear team list",
					)
					return nil
				}
				return output.ErrorWithHint(
					"NOT_FOUND",
					fmt.Sprintf("Team '%s' not found", teamKey),
					"Check available teams and use a valid team key",
					"linear team list",
				)
			}

			// Build input
			input := api.IssueCreateInput{
				Title:       title,
				TeamID:      team.ID,
				Description: description,
				ProjectID:   projectID,
				StateID:     stateID,
				ParentID:    parentID,
				DueDate:     dueDate,
				CycleID:     cycleID,
				ProjectMilestoneID: milestoneID,
			}

			if priority > 0 {
				input.Priority = &priority
			}

			if estimate > 0 {
				input.Estimate = &estimate
			}

			// Handle assignee
			if assignee != "" {
				if assignee == "self" || assignee == "me" {
					viewerID, err := client.GetViewerID(ctx)
					if err != nil {
						if IsHumanOutput() {
							output.ErrorHuman("Failed to get current user: " + err.Error())
							return nil
						}
						return output.Error("API_ERROR", "Failed to get current user: "+err.Error())
					}
					input.AssigneeID = viewerID
				} else {
					input.AssigneeID = assignee
				}
			}

			if len(labels) > 0 {
				input.LabelIDs = labels
			}

			result, err := client.CreateIssue(ctx, input)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			response := map[string]interface{}{
				"success": true,
				"issue": map[string]interface{}{
					"id":         result.ID,
					"identifier": result.Identifier,
					"url":        result.URL,
					"team": map[string]string{
						"key": result.TeamKey,
					},
				},
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Created issue %s: %s", result.Identifier, result.URL))
			} else {
				output.JSON(response)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "T", "", "Issue title (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Issue description (markdown)")
	cmd.Flags().IntVarP(&priority, "priority", "p", 0, "Priority (0=none, 1=urgent, 2=high, 3=medium, 4=low)")
	cmd.Flags().Float64VarP(&estimate, "estimate", "e", 0, "Story points estimate")
	cmd.Flags().StringVarP(&assignee, "assignee", "a", "", "Assignee (use 'self' for yourself, or user ID)")
	cmd.Flags().StringSliceVarP(&labels, "label", "l", nil, "Label IDs to apply")
	cmd.Flags().StringVar(&projectID, "project", "", "Project ID")
	cmd.Flags().StringVarP(&stateID, "state", "s", "", "Workflow state ID")
	cmd.Flags().StringVarP(&teamKey, "team", "t", "", "Team key (e.g., ENG)")
	cmd.Flags().StringVar(&parentID, "parent", "", "Parent issue ID for subtasks")
	cmd.Flags().StringVar(&dueDate, "due-date", "", "Due date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&cycleID, "cycle", "", "Cycle ID")
	cmd.Flags().StringVar(&milestoneID, "milestone", "", "Project milestone ID")

	return cmd
}

func newIssueUpdateCmd() *cobra.Command {
	var (
		title       string
		description string
		priority    int
		estimate    float64
		assignee    string
		labels      []string
		projectID   string
		stateID     string
		parentID    string
		dueDate     string
		cycleID     string
		milestoneID string
	)

	cmd := &cobra.Command{
		Use:   "update <issue-id>",
		Short: "Update an issue",
		Long: `Update an existing issue.

At least one field must be provided to update.

Examples:
  linear issue update ENG-123 --title "New title"
  linear issue update ENG-123 --priority 2
  linear issue update ENG-123 --assignee self --state abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]

			// Check that at least one field is provided
			if title == "" && description == "" && priority == 0 && estimate == 0 &&
				assignee == "" && len(labels) == 0 && projectID == "" && stateID == "" &&
				parentID == "" && dueDate == "" && cycleID == "" && milestoneID == "" {
				if IsHumanOutput() {
					output.ErrorHuman("At least one field must be provided to update")
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

			// Build input
			input := api.IssueUpdateInput{
				Title:              title,
				Description:        description,
				ProjectID:          projectID,
				StateID:            stateID,
				ParentID:           parentID,
				DueDate:            dueDate,
				CycleID:            cycleID,
				ProjectMilestoneID: milestoneID,
			}

			if priority > 0 {
				input.Priority = &priority
			}

			if estimate > 0 {
				input.Estimate = &estimate
			}

			// Handle assignee
			if assignee != "" {
				if assignee == "self" || assignee == "me" {
					viewerID, err := client.GetViewerID(ctx)
					if err != nil {
						if IsHumanOutput() {
							output.ErrorHuman("Failed to get current user: " + err.Error())
							return nil
						}
						return output.Error("API_ERROR", "Failed to get current user: "+err.Error())
					}
					input.AssigneeID = viewerID
				} else {
					input.AssigneeID = assignee
				}
			}

			if len(labels) > 0 {
				input.LabelIDs = labels
			}

			result, err := client.UpdateIssue(ctx, issueID, input)
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
				"issue": map[string]interface{}{
					"id":         result.ID,
					"identifier": result.Identifier,
					"url":        result.URL,
				},
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Updated issue %s", result.Identifier))
			} else {
				output.JSON(response)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "T", "", "New issue title")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New issue description (markdown)")
	cmd.Flags().IntVarP(&priority, "priority", "p", 0, "New priority (0=none, 1=urgent, 2=high, 3=medium, 4=low)")
	cmd.Flags().Float64VarP(&estimate, "estimate", "e", 0, "New story points estimate")
	cmd.Flags().StringVarP(&assignee, "assignee", "a", "", "New assignee (use 'self' for yourself, or user ID)")
	cmd.Flags().StringSliceVarP(&labels, "label", "l", nil, "Label IDs to apply (replaces existing)")
	cmd.Flags().StringVar(&projectID, "project", "", "New project ID")
	cmd.Flags().StringVarP(&stateID, "state", "s", "", "New workflow state ID")
	cmd.Flags().StringVar(&parentID, "parent", "", "New parent issue ID")
	cmd.Flags().StringVar(&dueDate, "due-date", "", "New due date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&cycleID, "cycle", "", "New cycle ID")
	cmd.Flags().StringVar(&milestoneID, "milestone", "", "New project milestone ID")

	return cmd
}

func newIssueDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <issue-id>",
		Short: "Delete an issue",
		Long: `Delete (trash) an issue.

Examples:
  linear issue delete ENG-123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			err = client.DeleteIssue(ctx, issueID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			response := map[string]interface{}{
				"success":   true,
				"operation": "delete",
				"issueId":   issueID,
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Deleted issue %s", issueID))
			} else {
				output.JSON(response)
			}

			return nil
		},
	}

	return cmd
}

func newIssueSearchCmd() *cobra.Command {
	var (
		limit           int
		includeArchived bool
		teamKey         string
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search issues",
		Long: `Search for issues by text.

Examples:
  linear issue search "authentication"
  linear issue search "bug fix" --limit 100
  linear issue search "old feature" --include-archived`,
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

			// Resolve team if provided
			var teamID string
			if teamKey != "" {
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

			results, err := client.SearchIssues(ctx, query, limit, includeArchived, teamID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				printSearchResultsHuman(results)
			} else {
				output.JSON(results)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Maximum number of results")
	cmd.Flags().BoolVar(&includeArchived, "include-archived", false, "Include archived issues")
	cmd.Flags().StringVarP(&teamKey, "team", "t", "", "Boost results from this team")

	return cmd
}

func newIssueRelateCmd() *cobra.Command {
	var (
		blocks      bool
		blockedBy   bool
		relatedTo   bool
		duplicateOf bool
	)

	cmd := &cobra.Command{
		Use:   "relate <issue-id> <related-id>",
		Short: "Create issue relationship",
		Long: `Create a relationship between two issues.

Relationship types (specify one):
  --blocks        Issue blocks the related issue
  --blocked-by    Issue is blocked by the related issue
  --related-to    Issues are related (default)
  --duplicate-of  Issue is a duplicate of the related issue

Examples:
  linear issue relate ENG-123 ENG-456 --blocks
  linear issue relate ENG-123 ENG-456 --related-to`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			relatedID := args[1]

			// Determine relationship type
			relationType := "related"
			if blocks {
				relationType = "blocks"
			} else if blockedBy {
				relationType = "blocked_by"
			} else if duplicateOf {
				relationType = "duplicate"
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

			err = client.CreateIssueRelation(ctx, issueID, relatedID, relationType)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			response := map[string]interface{}{
				"success":   true,
				"operation": "relate",
				"issueId":   issueID,
				"relatedId": relatedID,
				"type":      relationType,
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Created %s relationship between %s and %s", relationType, issueID, relatedID))
			} else {
				output.JSON(response)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&blocks, "blocks", false, "Issue blocks the related issue")
	cmd.Flags().BoolVar(&blockedBy, "blocked-by", false, "Issue is blocked by the related issue")
	cmd.Flags().BoolVar(&relatedTo, "related-to", false, "Issues are related (default)")
	cmd.Flags().BoolVar(&duplicateOf, "duplicate-of", false, "Issue is a duplicate of the related issue")

	return cmd
}

func newIssueUnrelateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unrelate <relation-id>",
		Short: "Remove issue relationship",
		Long: `Remove a relationship between issues.

Use 'issue relations <issue-id>' to find relation IDs.

Examples:
  linear issue unrelate abc123-relation-id`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			relationID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			err = client.DeleteIssueRelation(ctx, relationID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			response := map[string]interface{}{
				"success":    true,
				"operation":  "unrelate",
				"relationId": relationID,
			}

			if IsHumanOutput() {
				output.SuccessHuman("Removed issue relationship")
			} else {
				output.JSON(response)
			}

			return nil
		},
	}

	return cmd
}

func newIssueRelationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relations <issue-id>",
		Short: "View issue relationships",
		Long: `View all relationships for an issue.

Examples:
  linear issue relations ENG-123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			issue, err := client.GetIssue(ctx, issueID, false)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			response := map[string]interface{}{
				"issueId":   issue.ID,
				"identifier": issue.Identifier,
				"relations": issue.Relations,
				"count":     len(issue.Relations),
			}

			if IsHumanOutput() {
				printRelationsHuman(issue)
			} else {
				output.JSON(response)
			}

			return nil
		},
	}

	return cmd
}

func newIssueCommentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Manage issue comments",
	}

	cmd.AddCommand(newIssueCommentCreateCmd())
	cmd.AddCommand(newIssueCommentListCmd())

	return cmd
}

func newIssueCommentCreateCmd() *cobra.Command {
	var body string

	cmd := &cobra.Command{
		Use:   "create <issue-id>",
		Short: "Add a comment to an issue",
		Long: `Add a comment to an issue.

Examples:
  linear issue comment create ENG-123 --body "This is a comment"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]

			if body == "" {
				if IsHumanOutput() {
					output.ErrorHuman("Comment body is required. Use --body flag.")
					return nil
				}
				return output.Error("MISSING_BODY", "Comment body is required. Use --body flag.")
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

			comment, err := client.CreateComment(ctx, issueID, body)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			response := map[string]interface{}{
				"success":   true,
				"operation": "create",
				"comment":   comment,
			}

			if IsHumanOutput() {
				output.SuccessHuman("Comment added")
			} else {
				output.JSON(response)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&body, "body", "b", "", "Comment body (markdown)")

	return cmd
}

func newIssueCommentListCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "list <issue-id>",
		Short: "List comments on an issue",
		Long: `List all comments on an issue.

Examples:
  linear issue comment list ENG-123
  linear issue comment list ENG-123 --limit 100`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			comments, err := client.GetIssueComments(ctx, issueID, limit)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			response := map[string]interface{}{
				"comments": comments,
				"count":    len(comments),
			}

			if IsHumanOutput() {
				printCommentsHuman(comments)
			} else {
				output.JSON(response)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Maximum number of comments")

	return cmd
}

// Human output formatters

func printIssuesHuman(response *IssueListResponse, teamKey string) {
	if len(response.Issues) == 0 {
		output.HumanLn("No issues found for team %s", teamKey)
		return
	}

	output.HumanLn("Issues for team %s:\n", teamKey)

	headers := []string{"", "ID", "TITLE", "LABELS", "E", "A", "STATE", "UPDATED"}
	rows := make([][]string, len(response.Issues))

	for i, issue := range response.Issues {
		// Priority icon
		priorityIcon := display.PriorityIcon(issue.Priority)

		// Labels
		labelNames := make([]string, len(issue.Labels))
		for j, l := range issue.Labels {
			labelNames[j] = l.Name
		}
		labelsStr := strings.Join(labelNames, ", ")
		if len(labelsStr) > 20 {
			labelsStr = labelsStr[:17] + "..."
		}

		// Estimate
		estStr := ""
		if issue.Estimate != nil {
			estStr = fmt.Sprintf("%.0f", *issue.Estimate)
		}

		// Assignee
		assigneeStr := ""
		if issue.Assignee != nil {
			assigneeStr = display.Initials(issue.Assignee.DisplayName)
		}

		// Time ago
		updatedAt, _ := time.Parse(time.RFC3339, issue.UpdatedAt)
		timeAgo := display.TimeAgo(updatedAt)

		rows[i] = []string{
			priorityIcon,
			issue.Identifier,
			display.Truncate(issue.Title, 40),
			labelsStr,
			estStr,
			assigneeStr,
			issue.State.Name,
			output.Muted("%s", timeAgo),
		}
	}

	output.TableWithColors(headers, rows)
	output.HumanLn("\n%d issues", response.Count)
}

func printIssueDetailHuman(issue *api.IssueDetail) {
	output.HumanLn("%s %s", output.Bold("%s", issue.Identifier), issue.Title)
	output.HumanLn("%s", issue.URL)
	output.HumanLn("")

	// Metadata
	output.HumanLn("%s: %s", output.Bold("Status"), issue.State.Name)
	output.HumanLn("%s: %s", output.Bold("Team"), issue.Team.Name)

	if issue.Assignee != nil {
		output.HumanLn("%s: %s", output.Bold("Assignee"), issue.Assignee.DisplayName)
	}

	if issue.Priority > 0 {
		output.HumanLn("%s: %s %d", output.Bold("Priority"), display.PriorityIcon(issue.Priority), issue.Priority)
	}

	if issue.Estimate != nil {
		output.HumanLn("%s: %.0f", output.Bold("Estimate"), *issue.Estimate)
	}

	if issue.DueDate != "" {
		output.HumanLn("%s: %s", output.Bold("Due Date"), issue.DueDate)
	}

	if issue.Project != nil {
		output.HumanLn("%s: %s", output.Bold("Project"), issue.Project.Name)
	}

	if issue.ProjectMilestone != nil {
		output.HumanLn("%s: %s", output.Bold("Milestone"), issue.ProjectMilestone.Name)
	}

	if issue.Cycle != nil {
		output.HumanLn("%s: %s", output.Bold("Cycle"), issue.Cycle.Name)
	}

	if issue.Parent != nil {
		output.HumanLn("%s: %s - %s", output.Bold("Parent"), issue.Parent.Identifier, issue.Parent.Title)
	}

	if len(issue.Children) > 0 {
		output.HumanLn("%s:", output.Bold("Child Issues"))
		for _, child := range issue.Children {
			output.HumanLn("  • %s - %s (%s)", child.Identifier, child.Title, child.State.Name)
		}
	}

	if len(issue.Relations) > 0 {
		output.HumanLn("%s:", output.Bold("Relationships"))
		for _, rel := range issue.Relations {
			output.HumanLn("  • %s %s - %s", rel.Type, rel.RelatedIssue.Identifier, rel.RelatedIssue.Title)
		}
	}

	if len(issue.Labels) > 0 {
		labelNames := make([]string, len(issue.Labels))
		for i, l := range issue.Labels {
			labelNames[i] = l.Name
		}
		output.HumanLn("%s: %s", output.Bold("Labels"), strings.Join(labelNames, ", "))
	}

	createdAt, _ := time.Parse(time.RFC3339, issue.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, issue.UpdatedAt)
	output.HumanLn("%s: %s", output.Bold("Created"), display.TimeAgo(createdAt))
	output.HumanLn("%s: %s", output.Bold("Updated"), display.TimeAgo(updatedAt))

	// Description
	if issue.Description != "" {
		output.HumanLn("")
		output.HumanLn("%s", output.Bold("Description"))
		output.HumanLn("%s", issue.Description)
	}

	// Comments
	if len(issue.Comments) > 0 {
		output.HumanLn("")
		output.HumanLn("%s (%d)", output.Bold("Comments"), len(issue.Comments))
		for _, comment := range issue.Comments {
			author := "Unknown"
			if comment.User != nil {
				author = comment.User.DisplayName
			}
			createdAt, _ := time.Parse(time.RFC3339, comment.CreatedAt)
			output.HumanLn("")
			output.HumanLn("@%s commented %s", author, display.TimeAgo(createdAt))
			output.HumanLn("%s", comment.Body)
		}
	}
}

func printSearchResultsHuman(results *api.SearchIssuesResponse) {
	if len(results.Issues) == 0 {
		output.HumanLn("No issues found matching '%s'", results.Query)
		return
	}

	output.HumanLn("Search results for '%s':\n", results.Query)

	headers := []string{"ID", "TITLE", "STATE", "PRIORITY", "ASSIGNEE"}
	rows := make([][]string, len(results.Issues))

	for i, issue := range results.Issues {
		assigneeStr := output.Muted("Unassigned")
		if issue.Assignee != nil {
			assigneeStr = issue.Assignee.DisplayName
		}

		priorityStr := display.PriorityIcon(issue.Priority)

		rows[i] = []string{
			issue.Identifier,
			display.Truncate(issue.Title, 50),
			issue.State.Name,
			priorityStr,
			assigneeStr,
		}
	}

	output.TableWithColors(headers, rows)

	if results.HasMore {
		output.HumanLn("\n%d of %d issues (more available)", len(results.Issues), results.TotalCount)
	} else {
		output.HumanLn("\n%d issues", results.TotalCount)
	}
}

func printRelationsHuman(issue *api.IssueDetail) {
	if len(issue.Relations) == 0 {
		output.HumanLn("No relationships for %s", issue.Identifier)
		return
	}

	output.HumanLn("Relationships for %s:\n", issue.Identifier)

	headers := []string{"TYPE", "ISSUE", "TITLE", "RELATION ID"}
	rows := make([][]string, len(issue.Relations))

	for i, rel := range issue.Relations {
		rows[i] = []string{
			rel.Type,
			rel.RelatedIssue.Identifier,
			display.Truncate(rel.RelatedIssue.Title, 40),
			output.Muted("%s", rel.ID),
		}
	}

	output.TableWithColors(headers, rows)
}

func printCommentsHuman(comments []api.Comment) {
	if len(comments) == 0 {
		output.HumanLn("No comments")
		return
	}

	for _, comment := range comments {
		author := "Unknown"
		if comment.User != nil {
			author = comment.User.DisplayName
		}
		createdAt, _ := time.Parse(time.RFC3339, comment.CreatedAt)
		output.HumanLn("@%s commented %s", author, display.TimeAgo(createdAt))
		output.HumanLn("%s", comment.Body)
		output.HumanLn("")
	}

	output.HumanLn("%d comments", len(comments))
}

// Attachment commands

func newIssueAttachmentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "attachment",
		Aliases: []string{"attach"},
		Short:   "Manage issue attachments",
	}

	cmd.AddCommand(newIssueAttachmentCreateCmd())
	cmd.AddCommand(newIssueAttachmentListCmd())
	cmd.AddCommand(newIssueAttachmentDeleteCmd())

	return cmd
}

func newIssueAttachmentCreateCmd() *cobra.Command {
	var (
		title    string
		url      string
		subtitle string
	)

	cmd := &cobra.Command{
		Use:   "create <issue-id>",
		Short: "Add an attachment to an issue",
		Long: `Add a link attachment to an issue.

Attachments are external links (URLs) associated with an issue.

Examples:
  linear issue attachment create ENG-123 --title "Design Doc" --url "https://example.com/doc"
  linear issue attachment create ENG-123 -t "PR #42" -u "https://github.com/org/repo/pull/42" -s "Fixes the bug"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]

			if title == "" {
				if IsHumanOutput() {
					output.ErrorHuman("Attachment title is required. Use --title flag.")
					return nil
				}
				return output.Error("MISSING_TITLE", "Attachment title is required. Use --title flag.")
			}

			if url == "" {
				if IsHumanOutput() {
					output.ErrorHuman("Attachment URL is required. Use --url flag.")
					return nil
				}
				return output.Error("MISSING_URL", "Attachment URL is required. Use --url flag.")
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

			var subtitlePtr *string
			if subtitle != "" {
				subtitlePtr = &subtitle
			}

			attachment, err := client.CreateAttachment(ctx, issueID, title, url, subtitlePtr)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			response := map[string]interface{}{
				"success":    true,
				"operation":  "create",
				"attachment": attachment,
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Attachment added: %s", attachment.Title))
			} else {
				output.JSON(response)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Attachment title (required)")
	cmd.Flags().StringVarP(&url, "url", "u", "", "Attachment URL (required)")
	cmd.Flags().StringVarP(&subtitle, "subtitle", "s", "", "Attachment subtitle")

	return cmd
}

func newIssueAttachmentListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <issue-id>",
		Short: "List attachments for an issue",
		Long: `List all attachments for an issue.

Examples:
  linear issue attachment list ENG-123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			attachments, err := client.GetIssueAttachments(ctx, issueID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				printAttachmentsHuman(attachments, issueID)
			} else {
				output.JSON(attachments)
			}

			return nil
		},
	}

	return cmd
}

func newIssueAttachmentDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <attachment-id>",
		Short: "Delete an attachment",
		Long: `Delete an attachment from an issue.

Use 'linear issue attachment list <issue-id>' to find attachment IDs.

Examples:
  linear issue attachment delete abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			attachmentID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			err = client.DeleteAttachment(ctx, attachmentID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman("Attachment deleted")
			} else {
				output.JSON(map[string]interface{}{
					"success":      true,
					"operation":    "delete",
					"attachmentId": attachmentID,
				})
			}

			return nil
		},
	}

	return cmd
}

func printAttachmentsHuman(attachments *api.AttachmentsResponse, issueID string) {
	if len(attachments.Attachments) == 0 {
		output.HumanLn("No attachments for %s", issueID)
		return
	}

	output.HumanLn("Attachments for %s:\n", issueID)

	headers := []string{"TITLE", "URL", "CREATED", "ID"}
	rows := make([][]string, len(attachments.Attachments))

	for i, a := range attachments.Attachments {
		createdAt, _ := time.Parse(time.RFC3339, a.CreatedAt)
		rows[i] = []string{
			a.Title,
			display.Truncate(a.URL, 50),
			display.TimeAgo(createdAt),
			output.Muted("%s", a.ID),
		}
	}

	output.TableWithColors(headers, rows)
	output.HumanLn("\n%d attachments", attachments.Count)
}

// Issue utility commands

func newIssueStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <issue-id>",
		Short: "Start working on an issue",
		Long: `Mark an issue as started and optionally create a git branch.

This command:
  1. Updates the issue state to "started" (In Progress)
  2. Assigns the issue to you if unassigned
  3. Prints the suggested branch name

Examples:
  linear issue start ENG-123
  linear issue start ENG-123 --human`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			// Get current user for assignment
			viewer, err := client.GetViewer(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			// Get the issue first to find the "started" state
			issue, err := client.GetIssue(ctx, issueID, false)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if issue == nil {
				if IsHumanOutput() {
					output.ErrorHuman(fmt.Sprintf("Issue '%s' not found", issueID))
					return nil
				}
				return output.Error("NOT_FOUND", fmt.Sprintf("Issue '%s' not found", issueID))
			}

			// Get workflow states for the team
			states, err := client.GetWorkflowStates(ctx, issue.Team.ID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			// Find a "started" state
			var startedStateID string
			var startedStateName string
			for _, s := range states.WorkflowStates {
				if s.Type == "started" {
					startedStateID = s.ID
					startedStateName = s.Name
					break
				}
			}

			if startedStateID == "" {
				if IsHumanOutput() {
					output.ErrorHuman("No 'started' state found for this team")
					return nil
				}
				return output.Error("NO_STARTED_STATE", "No 'started' state found for this team")
			}

			// Update the issue
			updateInput := api.IssueUpdateInput{
				StateID: startedStateID,
			}

			// Assign to current user if unassigned
			if issue.Assignee == nil {
				updateInput.AssigneeID = viewer.Viewer.ID
			}

			result, err := client.UpdateIssue(ctx, issue.ID, updateInput)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			// Generate branch name
			branchName := generateBranchName(result.Identifier, issue.Title)

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Started %s: %s", result.Identifier, issue.Title))
				output.HumanLn("")
				output.HumanLn("State: %s", startedStateName)
				output.HumanLn("Assignee: %s", viewer.Viewer.DisplayName)
				output.HumanLn("")
				output.HumanLn("Suggested branch:")
				output.HumanLn("  git checkout -b %s", branchName)
			} else {
				output.JSON(map[string]interface{}{
					"success":    true,
					"operation":  "start",
					"identifier": result.Identifier,
					"title":      issue.Title,
					"state":      startedStateName,
					"assignee":   viewer.Viewer.DisplayName,
					"branchName": branchName,
					"url":        result.URL,
				})
			}

			return nil
		},
	}

	return cmd
}

func newIssueTitleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "title <issue-id>",
		Short: "Get issue title",
		Long: `Print the title of an issue.

Useful for scripts and commit messages.

Examples:
  linear issue title ENG-123
  git commit -m "$(linear issue title ENG-123)"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				return output.Error("AUTH_ERROR", err.Error())
			}

			issue, err := client.GetIssue(ctx, issueID, false)
			if err != nil {
				return output.Error("API_ERROR", err.Error())
			}

			if issue == nil {
				return output.Error("NOT_FOUND", fmt.Sprintf("Issue '%s' not found", issueID))
			}

			// Print without newline for scripting
			fmt.Print(issue.Title)
			return nil
		},
	}

	return cmd
}

func newIssueURLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "url <issue-id>",
		Short: "Get issue URL",
		Long: `Print the URL of an issue.

Useful for scripts and sharing.

Examples:
  linear issue url ENG-123
  open $(linear issue url ENG-123)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				return output.Error("AUTH_ERROR", err.Error())
			}

			issue, err := client.GetIssue(ctx, issueID, false)
			if err != nil {
				return output.Error("API_ERROR", err.Error())
			}

			if issue == nil {
				return output.Error("NOT_FOUND", fmt.Sprintf("Issue '%s' not found", issueID))
			}

			// Print without newline for scripting
			fmt.Print(issue.URL)
			return nil
		},
	}

	return cmd
}

func newIssueDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe <issue-id>",
		Short: "Print issue title with Linear-Issue trailer",
		Long: `Print the issue title followed by the Linear-Issue git trailer.

Useful for commit messages that link to Linear issues.

Examples:
  linear issue describe ENG-123
  git commit -m "$(linear issue describe ENG-123)"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				return output.Error("AUTH_ERROR", err.Error())
			}

			issue, err := client.GetIssue(ctx, issueID, false)
			if err != nil {
				return output.Error("API_ERROR", err.Error())
			}

			if issue == nil {
				return output.Error("NOT_FOUND", fmt.Sprintf("Issue '%s' not found", issueID))
			}

			// Print title with Linear-Issue trailer
			fmt.Printf("%s\n\nLinear-Issue: %s", issue.Title, issue.Identifier)
			return nil
		},
	}

	return cmd
}

// generateBranchName creates a git branch name from issue identifier and title
func generateBranchName(identifier, title string) string {
	// Lowercase the identifier
	branch := strings.ToLower(identifier)

	// Add slugified title
	slug := slugify(title)
	if slug != "" {
		branch = branch + "-" + slug
	}

	// Limit length
	if len(branch) > 50 {
		branch = branch[:50]
	}

	return branch
}

// slugify converts a string to a URL-safe slug
func slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces and underscores with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Remove non-alphanumeric characters (except hyphens)
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	// Remove consecutive hyphens
	slug := result.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Trim hyphens from ends
	slug = strings.Trim(slug, "-")

	return slug
}

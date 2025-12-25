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

// NewProjectCmd creates the project command group
func NewProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project",
		Aliases: []string{"p"},
		Short:   "Manage Linear projects",
		Long: `Create, view, update, and manage Linear projects.

Examples:
  linear project list
  linear project view <project-id>
  linear project create --name "Q1 Feature Development" --team ENG`,
	}

	cmd.AddCommand(newProjectListCmd())
	cmd.AddCommand(newProjectViewCmd())
	cmd.AddCommand(newProjectCreateCmd())
	cmd.AddCommand(newProjectUpdateCmd())
	cmd.AddCommand(newProjectDeleteCmd())
	cmd.AddCommand(newProjectRestoreCmd())
	cmd.AddCommand(newProjectMilestoneCmd())
	cmd.AddCommand(newProjectUpdateStatusCmd())

	return cmd
}

func newProjectListCmd() *cobra.Command {
	var (
		teamKey string
		limit   int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects",
		Long: `List projects with optional filters.

Examples:
  linear project list
  linear project list --team ENG
  linear project list --limit 20`,
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

			// Resolve team key to ID if provided
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
				if team == nil {
					if IsHumanOutput() {
						output.ErrorHuman(fmt.Sprintf("Team '%s' not found", teamKey))
						return nil
					}
					return output.Error("NOT_FOUND", fmt.Sprintf("Team '%s' not found", teamKey))
				}
				teamID = team.ID
			}

			projects, err := client.GetProjects(ctx, teamID, limit)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				printProjectsHuman(projects)
			} else {
				output.JSON(projects)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&teamKey, "team", "t", "", "Filter by team key (e.g., ENG)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 50, "Maximum projects to return")

	return cmd
}

func newProjectViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view <project-id>",
		Short: "View project details",
		Long: `View detailed information about a project.

Examples:
  linear project view abc123
  linear project view abc123 --human`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			project, err := client.GetProject(ctx, projectID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if project == nil {
				if IsHumanOutput() {
					output.ErrorHuman(fmt.Sprintf("Project '%s' not found", projectID))
					return nil
				}
				return output.Error("NOT_FOUND", fmt.Sprintf("Project '%s' not found", projectID))
			}

			if IsHumanOutput() {
				printProjectDetailHuman(project)
			} else {
				output.JSON(project)
			}

			return nil
		},
	}

	return cmd
}

func newProjectCreateCmd() *cobra.Command {
	var (
		name        string
		description string
		content     string
		teamKeys    []string
		statusID    string
		leadID      string
		icon        string
		color       string
		startDate   string
		targetDate  string
		priority    int
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new project",
		Long: `Create a new project in Linear.

Examples:
  linear project create --name "Q1 Feature Development" --team ENG
  linear project create --name "Auth Refactor" --team ENG --team BACKEND
  linear project create --name "Feature" --description "Description here" --target-date 2025-03-01`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				if IsHumanOutput() {
					output.ErrorHumanWithHint(
						"Project name is required",
						"Provide a name using the --name flag",
						"linear project create --name \"My Project\" --team ENG",
					)
					return nil
				}
				return output.ErrorWithHint(
					"MISSING_NAME",
					"Project name is required",
					"Provide a name using the --name flag",
					"linear project create --name \"My Project\" --team ENG",
				)
			}

			if len(teamKeys) == 0 {
				// Try default team
				defaultTeam := GetTeamID()
				if defaultTeam != "" {
					teamKeys = []string{defaultTeam}
				} else {
					if IsHumanOutput() {
						output.ErrorHumanWithHint(
							"At least one team is required",
							"Specify a team using --team flag or set a default team",
							"linear project create --name \"My Project\" --team ENG",
							"linear config set team_key ENG",
						)
						return nil
					}
					return output.ErrorWithHint(
						"MISSING_TEAM",
						"At least one team is required",
						"Specify a team using --team flag or set a default team",
						"linear project create --name \"My Project\" --team ENG",
						"linear config set team_key ENG",
					)
				}
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

			// Resolve team keys to IDs
			teamIDs := make([]string, 0, len(teamKeys))
			for _, key := range teamKeys {
				team, err := client.GetTeamByKey(ctx, key)
				if err != nil {
					if IsHumanOutput() {
						output.ErrorHuman(err.Error())
						return nil
					}
					return output.Error("API_ERROR", err.Error())
				}
				if team == nil {
					if IsHumanOutput() {
						output.ErrorHuman(fmt.Sprintf("Team '%s' not found", key))
						return nil
					}
					return output.Error("NOT_FOUND", fmt.Sprintf("Team '%s' not found", key))
				}
				teamIDs = append(teamIDs, team.ID)
			}

			input := api.ProjectCreateInput{
				Name:        name,
				Description: description,
				Content:     content,
				TeamIDs:     teamIDs,
				StatusID:    statusID,
				LeadID:      leadID,
				Icon:        icon,
				Color:       color,
				StartDate:   startDate,
				TargetDate:  targetDate,
			}

			if cmd.Flags().Changed("priority") {
				input.Priority = &priority
			}

			project, err := client.CreateProject(ctx, input)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Project created: %s", project.Name))
				output.HumanLn("  ID: %s", project.ID)
				output.HumanLn("  URL: %s", project.URL)
			} else {
				output.JSON(map[string]interface{}{
					"success":   true,
					"operation": "create",
					"project":   project,
				})
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Project name (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Project description")
	cmd.Flags().StringVar(&content, "content", "", "Project content (markdown)")
	cmd.Flags().StringArrayVarP(&teamKeys, "team", "t", nil, "Team key (can be specified multiple times)")
	cmd.Flags().StringVar(&statusID, "status-id", "", "Project status ID")
	cmd.Flags().StringVar(&leadID, "lead", "", "Project lead user ID")
	cmd.Flags().StringVar(&icon, "icon", "", "Project icon")
	cmd.Flags().StringVar(&color, "color", "", "Project color (#RRGGBB)")
	cmd.Flags().StringVar(&startDate, "start-date", "", "Project start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&targetDate, "target-date", "", "Project target date (YYYY-MM-DD)")
	cmd.Flags().IntVar(&priority, "priority", 0, "Project priority (0-4)")

	return cmd
}

func newProjectUpdateCmd() *cobra.Command {
	var (
		name        string
		description string
		content     string
		statusID    string
		leadID      string
		icon        string
		color       string
		startDate   string
		targetDate  string
		priority    int
	)

	cmd := &cobra.Command{
		Use:   "update <project-id>",
		Short: "Update a project",
		Long: `Update an existing project.

Examples:
  linear project update abc123 --name "New Name"
  linear project update abc123 --description "Updated description"
  linear project update abc123 --target-date 2025-06-01`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]

			// Check if at least one field is being updated
			if !cmd.Flags().Changed("name") &&
				!cmd.Flags().Changed("description") &&
				!cmd.Flags().Changed("content") &&
				!cmd.Flags().Changed("status-id") &&
				!cmd.Flags().Changed("lead") &&
				!cmd.Flags().Changed("icon") &&
				!cmd.Flags().Changed("color") &&
				!cmd.Flags().Changed("start-date") &&
				!cmd.Flags().Changed("target-date") &&
				!cmd.Flags().Changed("priority") {
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

			input := api.ProjectUpdateInput{}

			if cmd.Flags().Changed("name") {
				input.Name = name
			}
			if cmd.Flags().Changed("description") {
				input.Description = description
			}
			if cmd.Flags().Changed("content") {
				input.Content = content
			}
			if cmd.Flags().Changed("status-id") {
				input.StatusID = statusID
			}
			if cmd.Flags().Changed("lead") {
				input.LeadID = leadID
			}
			if cmd.Flags().Changed("icon") {
				input.Icon = icon
			}
			if cmd.Flags().Changed("color") {
				input.Color = color
			}
			if cmd.Flags().Changed("start-date") {
				input.StartDate = startDate
			}
			if cmd.Flags().Changed("target-date") {
				input.TargetDate = targetDate
			}
			if cmd.Flags().Changed("priority") {
				input.Priority = &priority
			}

			project, err := client.UpdateProject(ctx, projectID, input)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Project updated: %s", project.Name))
			} else {
				output.JSON(map[string]interface{}{
					"success":   true,
					"operation": "update",
					"project":   project,
				})
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Project name")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Project description")
	cmd.Flags().StringVar(&content, "content", "", "Project content (markdown)")
	cmd.Flags().StringVar(&statusID, "status-id", "", "Project status ID")
	cmd.Flags().StringVar(&leadID, "lead", "", "Project lead user ID")
	cmd.Flags().StringVar(&icon, "icon", "", "Project icon")
	cmd.Flags().StringVar(&color, "color", "", "Project color (#RRGGBB)")
	cmd.Flags().StringVar(&startDate, "start-date", "", "Project start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&targetDate, "target-date", "", "Project target date (YYYY-MM-DD)")
	cmd.Flags().IntVar(&priority, "priority", 0, "Project priority (0-4)")

	return cmd
}

func newProjectDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <project-id>",
		Short: "Delete (archive) a project",
		Long: `Delete (archive) a project. The project can be restored later.

Examples:
  linear project delete abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			err = client.DeleteProject(ctx, projectID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman("Project deleted")
			} else {
				output.JSON(map[string]interface{}{
					"success":   true,
					"operation": "delete",
					"projectId": projectID,
				})
			}

			return nil
		},
	}

	return cmd
}

func newProjectRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <project-id>",
		Short: "Restore a deleted project",
		Long: `Restore a previously deleted (archived) project.

Examples:
  linear project restore abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			err = client.RestoreProject(ctx, projectID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman("Project restored")
			} else {
				output.JSON(map[string]interface{}{
					"success":   true,
					"operation": "restore",
					"projectId": projectID,
				})
			}

			return nil
		},
	}

	return cmd
}

func newProjectMilestoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "milestone",
		Short: "Manage project milestones",
		Long: `Create, list, update, and delete project milestones.

Examples:
  linear project milestone list <project-id>
  linear project milestone create <project-id> --name "Beta Release"`,
	}

	cmd.AddCommand(newProjectMilestoneListCmd())
	cmd.AddCommand(newProjectMilestoneCreateCmd())
	cmd.AddCommand(newProjectMilestoneUpdateCmd())
	cmd.AddCommand(newProjectMilestoneDeleteCmd())

	return cmd
}

func newProjectMilestoneListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <project-id>",
		Short: "List milestones for a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			milestones, err := client.GetProjectMilestones(ctx, projectID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				printMilestonesHuman(milestones)
			} else {
				output.JSON(milestones)
			}

			return nil
		},
	}
}

func newProjectMilestoneCreateCmd() *cobra.Command {
	var (
		name        string
		description string
		targetDate  string
	)

	cmd := &cobra.Command{
		Use:   "create <project-id>",
		Short: "Create a milestone",
		Long: `Create a new milestone for a project.

Examples:
  linear project milestone create abc123 --name "Beta Release"
  linear project milestone create abc123 --name "v1.0" --target-date 2025-03-01`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]

			if name == "" {
				if IsHumanOutput() {
					output.ErrorHuman("Milestone name is required. Use --name flag.")
					return nil
				}
				return output.Error("MISSING_NAME", "Milestone name is required")
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

			milestone, err := client.CreateProjectMilestone(ctx, projectID, name, description, targetDate)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Milestone created: %s", milestone.Name))
			} else {
				output.JSON(map[string]interface{}{
					"success":   true,
					"operation": "create",
					"milestone": milestone,
				})
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Milestone name (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Milestone description")
	cmd.Flags().StringVar(&targetDate, "target-date", "", "Target date (YYYY-MM-DD)")

	return cmd
}

func newProjectMilestoneUpdateCmd() *cobra.Command {
	var (
		name        string
		description string
		targetDate  string
	)

	cmd := &cobra.Command{
		Use:   "update <milestone-id>",
		Short: "Update a milestone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			milestoneID := args[0]

			if !cmd.Flags().Changed("name") &&
				!cmd.Flags().Changed("description") &&
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

			var namePtr, descPtr, datePtr *string
			if cmd.Flags().Changed("name") {
				namePtr = &name
			}
			if cmd.Flags().Changed("description") {
				descPtr = &description
			}
			if cmd.Flags().Changed("target-date") {
				datePtr = &targetDate
			}

			milestone, err := client.UpdateProjectMilestone(ctx, milestoneID, namePtr, descPtr, datePtr)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Milestone updated: %s", milestone.Name))
			} else {
				output.JSON(map[string]interface{}{
					"success":   true,
					"operation": "update",
					"milestone": milestone,
				})
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Milestone name")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Milestone description")
	cmd.Flags().StringVar(&targetDate, "target-date", "", "Target date (YYYY-MM-DD)")

	return cmd
}

func newProjectMilestoneDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <milestone-id>",
		Short: "Delete a milestone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			milestoneID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			err = client.DeleteProjectMilestone(ctx, milestoneID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman("Milestone deleted")
			} else {
				output.JSON(map[string]interface{}{
					"success":     true,
					"operation":   "delete",
					"milestoneId": milestoneID,
				})
			}

			return nil
		},
	}
}

func newProjectUpdateStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update-status",
		Aliases: []string{"updates"},
		Short:   "Manage project status updates",
		Long: `Create and list project status updates (changelog).

Examples:
  linear project update-status list <project-id>
  linear project update-status create <project-id> --body "Progress update"`,
	}

	cmd.AddCommand(newProjectUpdateStatusListCmd())
	cmd.AddCommand(newProjectUpdateStatusCreateCmd())

	return cmd
}

func newProjectUpdateStatusListCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "list <project-id>",
		Short: "List status updates for a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			ctx := context.Background()

			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("AUTH_ERROR", err.Error())
			}

			updates, err := client.GetProjectUpdates(ctx, projectID, limit)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				printProjectUpdatesHuman(updates)
			} else {
				output.JSON(updates)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Maximum updates to return")

	return cmd
}

func newProjectUpdateStatusCreateCmd() *cobra.Command {
	var (
		body   string
		health string
	)

	cmd := &cobra.Command{
		Use:   "create <project-id>",
		Short: "Create a status update",
		Long: `Create a new status update for a project.

Health values: onTrack, atRisk, offTrack

Examples:
  linear project update-status create abc123 --body "All tasks completed for sprint 1"
  linear project update-status create abc123 --body "Delayed due to dependencies" --health atRisk`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]

			if body == "" {
				if IsHumanOutput() {
					output.ErrorHuman("Update body is required. Use --body flag.")
					return nil
				}
				return output.Error("MISSING_BODY", "Update body is required")
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

			var healthPtr *string
			if health != "" {
				healthPtr = &health
			}

			update, err := client.CreateProjectUpdate(ctx, projectID, body, healthPtr)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman("Status update created")
				output.HumanLn("  ID: %s", update.ID)
			} else {
				output.JSON(map[string]interface{}{
					"success":   true,
					"operation": "create",
					"update":    update,
				})
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&body, "body", "b", "", "Update body (required)")
	cmd.Flags().StringVar(&health, "health", "", "Project health (onTrack, atRisk, offTrack)")

	return cmd
}

// Human output formatters

func printProjectsHuman(projects *api.ProjectsResponse) {
	if len(projects.Projects) == 0 {
		output.HumanLn("No projects found")
		return
	}

	headers := []string{"NAME", "STATUS", "PROGRESS", "LEAD", "TEAMS", "TARGET", "ID"}
	rows := make([][]string, len(projects.Projects))

	for i, p := range projects.Projects {
		statusName := "-"
		if p.Status != nil {
			statusName = p.Status.Name
		}

		leadName := "-"
		if p.Lead != nil {
			leadName = p.Lead.DisplayName
		}

		teamKeys := make([]string, len(p.Teams))
		for j, t := range p.Teams {
			teamKeys[j] = t.Key
		}
		teamsStr := strings.Join(teamKeys, ", ")
		if teamsStr == "" {
			teamsStr = "-"
		}

		targetDate := "-"
		if p.TargetDate != "" {
			targetDate = p.TargetDate
		}

		progress := fmt.Sprintf("%.0f%%", p.Progress*100)

		rows[i] = []string{
			display.Truncate(p.Name, 40),
			statusName,
			progress,
			leadName,
			teamsStr,
			targetDate,
			output.Muted("%s", p.ID),
		}
	}

	output.TableWithColors(headers, rows)
	output.HumanLn("\n%d projects", projects.Count)
}

func printProjectDetailHuman(p *api.ProjectDetail) {
	output.HumanLn("%s", p.Name)
	output.HumanLn("")

	if p.Description != "" {
		output.HumanLn("Description: %s", p.Description)
	}

	if p.Status != nil {
		output.HumanLn("Status: %s", p.Status.Name)
	}

	output.HumanLn("Progress: %.0f%%", p.Progress*100)

	if p.Lead != nil {
		output.HumanLn("Lead: %s", p.Lead.DisplayName)
	}

	if len(p.Teams) > 0 {
		teamNames := make([]string, len(p.Teams))
		for i, t := range p.Teams {
			teamNames[i] = t.Key
		}
		output.HumanLn("Teams: %s", strings.Join(teamNames, ", "))
	}

	if p.StartDate != "" {
		output.HumanLn("Start Date: %s", p.StartDate)
	}

	if p.TargetDate != "" {
		output.HumanLn("Target Date: %s", p.TargetDate)
	}

	output.HumanLn("")
	output.HumanLn("URL: %s", p.URL)
	output.HumanLn("ID: %s", output.Muted("%s", p.ID))

	if p.Content != "" {
		output.HumanLn("")
		output.HumanLn("Content:")
		output.HumanLn("%s", p.Content)
	}
}

func printMilestonesHuman(milestones *api.MilestonesResponse) {
	if len(milestones.Milestones) == 0 {
		output.HumanLn("No milestones found")
		return
	}

	headers := []string{"NAME", "TARGET DATE", "ID"}
	rows := make([][]string, len(milestones.Milestones))

	for i, m := range milestones.Milestones {
		targetDate := "-"
		if m.TargetDate != "" {
			targetDate = m.TargetDate
		}

		rows[i] = []string{
			m.Name,
			targetDate,
			output.Muted("%s", m.ID),
		}
	}

	output.TableWithColors(headers, rows)
	output.HumanLn("\n%d milestones", milestones.Count)
}

func printProjectUpdatesHuman(updates *api.ProjectUpdatesResponse) {
	if len(updates.Updates) == 0 {
		output.HumanLn("No status updates found")
		return
	}

	for _, u := range updates.Updates {
		createdAt := u.CreatedAt
		if t, err := time.Parse(time.RFC3339, u.CreatedAt); err == nil {
			createdAt = display.TimeAgo(t)
		}

		healthStr := ""
		if u.Health != "" {
			healthStr = fmt.Sprintf(" [%s]", u.Health)
		}

		userName := "Unknown"
		if u.User != nil {
			userName = u.User.DisplayName
		}

		output.HumanLn("%s by %s%s", createdAt, userName, healthStr)
		output.HumanLn("  %s", display.Truncate(u.Body, 80))
		output.HumanLn("")
	}

	output.HumanLn("%d updates", updates.Count)
}

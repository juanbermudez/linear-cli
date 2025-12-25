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

// NewWorkflowCmd creates the workflow command group
func NewWorkflowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workflow",
		Aliases: []string{"w"},
		Short:   "Manage workflow states",
		Long: `List and manage workflow states (issue statuses) for a team.

Workflow states are cached for 24 hours. Use 'workflow cache' to refresh.

Examples:
  linear workflow list --team ENG
  linear workflow cache --team ENG`,
	}

	cmd.AddCommand(newWorkflowListCmd())
	cmd.AddCommand(newWorkflowCacheCmd())

	return cmd
}

func newWorkflowListCmd() *cobra.Command {
	var (
		teamKey string
		refresh bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workflow states for a team",
		Long: `List all workflow states (issue statuses) for a team.

States include: triage, backlog, unstarted, started, completed, canceled.
Results are cached for 24 hours.

Examples:
  linear workflow list --team ENG
  linear workflow list --team ENG --refresh
  linear workflow list --team ENG --human`,
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

			// Resolve team key to ID if needed
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

			var states *api.WorkflowStatesResponse

			// Try cache first
			cacheManager, _ := cache.NewManager()
			cacheKey := cache.TeamKey("workflows", team.ID)

			if !refresh && cacheManager != nil {
				cached, _ := cache.Read[api.WorkflowStatesResponse](cacheManager, cacheKey)
				if cached != nil {
					states = cached
				}
			}

			// Fetch if not cached
			if states == nil {
				states, err = client.GetWorkflowStates(ctx, team.ID)
				if err != nil {
					if IsHumanOutput() {
						output.ErrorHuman(err.Error())
						return nil
					}
					return output.Error("API_ERROR", err.Error())
				}

				// Cache the results
				if cacheManager != nil {
					cache.Write(cacheManager, cacheKey, *states)
				}
			}

			// Sort by position
			sort.Slice(states.WorkflowStates, func(i, j int) bool {
				return states.WorkflowStates[i].Position < states.WorkflowStates[j].Position
			})

			if IsHumanOutput() {
				printWorkflowStatesHuman(states, team.Key)
			} else {
				output.JSON(states)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&teamKey, "team", "t", "", "Team key (e.g., ENG)")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Bypass cache and fetch fresh data")

	return cmd
}

func newWorkflowCacheCmd() *cobra.Command {
	var teamKey string

	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Force refresh workflow state cache",
		Long: `Force refresh the workflow state cache for a team.

This fetches fresh data from Linear and updates the local cache.

Examples:
  linear workflow cache --team ENG`,
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

			// Fetch fresh data
			states, err := client.GetWorkflowStates(ctx, team.ID)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			// Update cache
			cacheManager, _ := cache.NewManager()
			if cacheManager != nil {
				cacheKey := cache.TeamKey("workflows", team.ID)
				cache.Write(cacheManager, cacheKey, *states)
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Cached %d workflow states for team %s", states.Count, team.Key))
			} else {
				output.JSON(map[string]interface{}{
					"success": true,
					"message": fmt.Sprintf("Cached %d workflow states", states.Count),
					"team":    team.Key,
					"count":   states.Count,
				})
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&teamKey, "team", "t", "", "Team key (e.g., ENG)")

	return cmd
}

func printWorkflowStatesHuman(states *api.WorkflowStatesResponse, teamKey string) {
	if len(states.WorkflowStates) == 0 {
		output.HumanLn("No workflow states found for team %s", teamKey)
		return
	}

	output.HumanLn("Workflow states for team %s:\n", teamKey)

	headers := []string{"TYPE", "NAME", "POSITION", "ID"}
	rows := make([][]string, len(states.WorkflowStates))

	for i, s := range states.WorkflowStates {
		typeDisplay := formatWorkflowType(s.Type)
		rows[i] = []string{
			typeDisplay,
			s.Name,
			fmt.Sprintf("%d", s.Position),
			output.Muted("%s", s.ID),
		}
	}

	output.TableWithColors(headers, rows)
	output.HumanLn("\n%d states", states.Count)
}

func formatWorkflowType(stateType string) string {
	icon := display.StatusIcon(stateType)
	return fmt.Sprintf("%s %s", icon, stateType)
}

package cmd

import (
	"context"
	"fmt"
	"sort"

	"github.com/juanbermudez/agent-linear-cli/internal/api"
	"github.com/juanbermudez/agent-linear-cli/internal/cache"
	"github.com/juanbermudez/agent-linear-cli/internal/output"
	"github.com/spf13/cobra"
)

// ProjectStatusesResponse is the response for project statuses
type ProjectStatusesResponse struct {
	ProjectStatuses []ProjectStatus `json:"projectStatuses"`
	Count           int             `json:"count"`
}

// ProjectStatus represents a project status
type ProjectStatus struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Position int    `json:"position"`
}

// NewStatusCmd creates the status command group
func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Manage project statuses",
		Long: `List and manage project statuses.

Project statuses are workspace-wide and include states like:
planned, backlog, started, paused, completed, canceled.

Examples:
  linear status list
  linear status cache`,
	}

	cmd.AddCommand(newStatusListCmd())
	cmd.AddCommand(newStatusCacheCmd())

	return cmd
}

func newStatusListCmd() *cobra.Command {
	var refresh bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List project statuses",
		Long: `List all project statuses in your workspace.

Project statuses are used for tracking project progress.
Results are cached for 24 hours.

Examples:
  linear status list
  linear status list --refresh
  linear status list --human`,
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

			var statuses *ProjectStatusesResponse

			// Try cache first
			cacheManager, _ := cache.NewManager()
			cacheKey := cache.WorkspaceKey("statuses")

			if !refresh && cacheManager != nil {
				cached, _ := cache.Read[ProjectStatusesResponse](cacheManager, cacheKey)
				if cached != nil {
					statuses = cached
				}
			}

			// Fetch if not cached
			if statuses == nil {
				statuses, err = fetchProjectStatuses(ctx, client)
				if err != nil {
					if IsHumanOutput() {
						output.ErrorHuman(err.Error())
						return nil
					}
					return output.Error("API_ERROR", err.Error())
				}

				// Cache the results
				if cacheManager != nil {
					cache.Write(cacheManager, cacheKey, *statuses)
				}
			}

			// Sort by position
			sort.Slice(statuses.ProjectStatuses, func(i, j int) bool {
				return statuses.ProjectStatuses[i].Position < statuses.ProjectStatuses[j].Position
			})

			if IsHumanOutput() {
				printProjectStatusesHuman(statuses)
			} else {
				output.JSON(statuses)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&refresh, "refresh", false, "Bypass cache and fetch fresh data")

	return cmd
}

func newStatusCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Force refresh project status cache",
		Long: `Force refresh the project status cache.

This fetches fresh data from Linear and updates the local cache.

Examples:
  linear status cache`,
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

			// Fetch fresh data
			statuses, err := fetchProjectStatuses(ctx, client)
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
				cacheKey := cache.WorkspaceKey("statuses")
				cache.Write(cacheManager, cacheKey, *statuses)
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Cached %d project statuses", statuses.Count))
			} else {
				output.JSON(map[string]interface{}{
					"success": true,
					"message": fmt.Sprintf("Cached %d project statuses", statuses.Count),
					"count":   statuses.Count,
				})
			}

			return nil
		},
	}

	return cmd
}

// fetchProjectStatuses fetches project statuses from the API
// Linear has a fixed set of project statuses
func fetchProjectStatuses(ctx context.Context, client *api.Client) (*ProjectStatusesResponse, error) {
	// Project statuses in Linear are fixed/predefined
	// We query the organization for available project statuses
	var query struct {
		ProjectStatuses []struct {
			ID          string  `graphql:"id"`
			Name        string  `graphql:"name"`
			Type        string  `graphql:"type"`
			Position    float64 `graphql:"position"`
			Description string  `graphql:"description"`
		} `graphql:"projectStatuses"`
	}

	if err := client.Query(ctx, &query, nil); err != nil {
		return nil, err
	}

	statuses := make([]ProjectStatus, len(query.ProjectStatuses))
	for i, s := range query.ProjectStatuses {
		statuses[i] = ProjectStatus{
			ID:       s.ID,
			Name:     s.Name,
			Type:     s.Type,
			Position: int(s.Position),
		}
	}

	return &ProjectStatusesResponse{
		ProjectStatuses: statuses,
		Count:           len(statuses),
	}, nil
}

func printProjectStatusesHuman(statuses *ProjectStatusesResponse) {
	if len(statuses.ProjectStatuses) == 0 {
		output.HumanLn("No project statuses found")
		return
	}

	headers := []string{"TYPE", "NAME", "POSITION", "ID"}
	rows := make([][]string, len(statuses.ProjectStatuses))

	for i, s := range statuses.ProjectStatuses {
		rows[i] = []string{
			s.Type,
			s.Name,
			fmt.Sprintf("%d", s.Position),
			output.Muted("%s", s.ID),
		}
	}

	output.TableWithColors(headers, rows)
	output.HumanLn("\n%d statuses", statuses.Count)
}

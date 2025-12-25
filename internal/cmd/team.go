package cmd

import (
	"context"
	"sort"

	"github.com/juanbermudez/agent-linear-cli/internal/api"
	"github.com/juanbermudez/agent-linear-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewTeamCmd creates the team command group
func NewTeamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "team",
		Aliases: []string{"t"},
		Short:   "Manage Linear teams",
		Long: `List and manage Linear teams in your workspace.

Examples:
  linear team list
  linear team list --human`,
	}

	cmd.AddCommand(newTeamListCmd())

	return cmd
}

func newTeamListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all teams",
		Long: `List all teams in your Linear workspace.

Output includes team key, name, and ID. Teams are sorted alphabetically by name.
Archived teams are excluded.

Examples:
  linear team list
  linear team list --human`,
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

			teams, err := client.GetTeams(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			// Sort teams alphabetically by name
			sort.Slice(teams.Teams, func(i, j int) bool {
				return teams.Teams[i].Name < teams.Teams[j].Name
			})

			if IsHumanOutput() {
				printTeamsHuman(teams)
			} else {
				output.JSON(teams)
			}

			return nil
		},
	}

	return cmd
}

func printTeamsHuman(teams *api.TeamsResponse) {
	if len(teams.Teams) == 0 {
		output.HumanLn("No teams found")
		return
	}

	headers := []string{"KEY", "NAME", "ID"}
	rows := make([][]string, len(teams.Teams))

	for i, t := range teams.Teams {
		rows[i] = []string{
			t.Key,
			t.Name,
			output.Muted("%s", t.ID),
		}
	}

	output.TableWithColors(headers, rows)
	output.HumanLn("\n%d teams", teams.Count)
}

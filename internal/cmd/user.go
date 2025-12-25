package cmd

import (
	"context"
	"sort"
	"strings"

	"github.com/juanbermudez/agent-linear-cli/internal/api"
	"github.com/juanbermudez/agent-linear-cli/internal/cache"
	"github.com/juanbermudez/agent-linear-cli/internal/display"
	"github.com/juanbermudez/agent-linear-cli/internal/output"
	"github.com/spf13/cobra"
)

// UserListResponse is the response for user list command
type UserListResponse struct {
	Users []api.User `json:"users"`
	Count int        `json:"count"`
}

// UserSearchResponse is the response for user search command
type UserSearchResponse struct {
	Users []api.User `json:"users"`
	Count int        `json:"count"`
	Query string     `json:"query"`
}

// NewUserCmd creates the user command group
func NewUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "user",
		Aliases: []string{"u"},
		Short:   "Manage Linear users",
		Long: `List and search users in your Linear workspace.

Examples:
  linear user list
  linear user search "john"`,
	}

	cmd.AddCommand(newUserListCmd())
	cmd.AddCommand(newUserSearchCmd())

	return cmd
}

func newUserListCmd() *cobra.Command {
	var (
		activeOnly bool
		adminsOnly bool
		refresh    bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all users",
		Long: `List all users in your Linear workspace.

Users are cached for 24 hours. Use --refresh to bypass cache.

Examples:
  linear user list
  linear user list --active-only
  linear user list --admins-only
  linear user list --refresh`,
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

			var users *api.UsersResponse

			// Try cache first (unless refresh requested)
			cacheManager, _ := cache.NewManager()
			cacheKey := cache.WorkspaceKey("users")

			if !refresh && cacheManager != nil {
				cached, _ := cache.Read[api.UsersResponse](cacheManager, cacheKey)
				if cached != nil {
					users = cached
				}
			}

			// Fetch if not cached
			if users == nil {
				users, err = client.GetUsers(ctx)
				if err != nil {
					if IsHumanOutput() {
						output.ErrorHuman(err.Error())
						return nil
					}
					return output.Error("API_ERROR", err.Error())
				}

				// Cache the results
				if cacheManager != nil {
					cache.Write(cacheManager, cacheKey, *users)
				}
			}

			// Apply filters
			filteredUsers := filterUsers(users.Users, activeOnly, adminsOnly)

			// Sort by display name
			sort.Slice(filteredUsers, func(i, j int) bool {
				return filteredUsers[i].DisplayName < filteredUsers[j].DisplayName
			})

			response := &UserListResponse{
				Users: filteredUsers,
				Count: len(filteredUsers),
			}

			if IsHumanOutput() {
				printUsersHuman(response)
			} else {
				output.JSON(response)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&activeOnly, "active-only", false, "Show only active users")
	cmd.Flags().BoolVar(&adminsOnly, "admins-only", false, "Show only admin users")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Bypass cache and fetch fresh data")

	return cmd
}

func newUserSearchCmd() *cobra.Command {
	var (
		activeOnly bool
		refresh    bool
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search users by name or email",
		Long: `Search for users by name, display name, or email.

The search is case-insensitive and matches partial strings.

Examples:
  linear user search "john"
  linear user search "example.com"
  linear user search "john" --active-only`,
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

			var users *api.UsersResponse

			// Try cache first
			cacheManager, _ := cache.NewManager()
			cacheKey := cache.WorkspaceKey("users")

			if !refresh && cacheManager != nil {
				cached, _ := cache.Read[api.UsersResponse](cacheManager, cacheKey)
				if cached != nil {
					users = cached
				}
			}

			// Fetch if not cached
			if users == nil {
				users, err = client.GetUsers(ctx)
				if err != nil {
					if IsHumanOutput() {
						output.ErrorHuman(err.Error())
						return nil
					}
					return output.Error("API_ERROR", err.Error())
				}

				// Cache the results
				if cacheManager != nil {
					cache.Write(cacheManager, cacheKey, *users)
				}
			}

			// Search and filter
			matchedUsers := searchUsers(users.Users, query)
			if activeOnly {
				matchedUsers = filterUsers(matchedUsers, true, false)
			}

			// Sort by display name
			sort.Slice(matchedUsers, func(i, j int) bool {
				return matchedUsers[i].DisplayName < matchedUsers[j].DisplayName
			})

			response := &UserSearchResponse{
				Users: matchedUsers,
				Count: len(matchedUsers),
				Query: query,
			}

			if IsHumanOutput() {
				printUserSearchHuman(response)
			} else {
				output.JSON(response)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&activeOnly, "active-only", false, "Show only active users")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Bypass cache and fetch fresh data")

	return cmd
}

func filterUsers(users []api.User, activeOnly, adminsOnly bool) []api.User {
	if !activeOnly && !adminsOnly {
		return users
	}

	filtered := make([]api.User, 0)
	for _, u := range users {
		if activeOnly && !u.Active {
			continue
		}
		if adminsOnly && !u.Admin {
			continue
		}
		filtered = append(filtered, u)
	}
	return filtered
}

func searchUsers(users []api.User, query string) []api.User {
	query = strings.ToLower(query)
	matched := make([]api.User, 0)

	for _, u := range users {
		if strings.Contains(strings.ToLower(u.Name), query) ||
			strings.Contains(strings.ToLower(u.DisplayName), query) ||
			strings.Contains(strings.ToLower(u.Email), query) {
			matched = append(matched, u)
		}
	}

	return matched
}

func printUsersHuman(response *UserListResponse) {
	if len(response.Users) == 0 {
		output.HumanLn("No users found")
		return
	}

	headers := []string{"NAME", "EMAIL", "STATUS", "ADMIN", "ID"}
	rows := make([][]string, len(response.Users))

	for i, u := range response.Users {
		status := "Active"
		if !u.Active {
			status = output.Muted("Inactive")
		}

		admin := ""
		if u.Admin {
			admin = display.BoolToCheckmark(u.Admin)
		}

		rows[i] = []string{
			u.DisplayName,
			u.Email,
			status,
			admin,
			output.Muted("%s", u.ID),
		}
	}

	output.TableWithColors(headers, rows)
	output.HumanLn("\n%d users", response.Count)
}

func printUserSearchHuman(response *UserSearchResponse) {
	if len(response.Users) == 0 {
		output.HumanLn("No users found matching '%s'", response.Query)
		return
	}

	output.HumanLn("Search results for '%s':\n", response.Query)

	headers := []string{"NAME", "EMAIL", "STATUS", "ADMIN", "ID"}
	rows := make([][]string, len(response.Users))

	for i, u := range response.Users {
		status := "Active"
		if !u.Active {
			status = output.Muted("Inactive")
		}

		admin := ""
		if u.Admin {
			admin = display.BoolToCheckmark(u.Admin)
		}

		rows[i] = []string{
			u.DisplayName,
			u.Email,
			status,
			admin,
			output.Muted("%s", u.ID),
		}
	}

	output.TableWithColors(headers, rows)
	output.HumanLn("\n%d users found", response.Count)
}

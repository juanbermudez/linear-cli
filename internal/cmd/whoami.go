package cmd

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/juanbermudez/agent-linear-cli/internal/api"
	"github.com/juanbermudez/agent-linear-cli/internal/auth"
	"github.com/spf13/cobra"
)

// WhoamiResponse represents the whoami command output
type WhoamiResponse struct {
	User         *api.Viewer       `json:"user"`
	Organization *api.Organization `json:"organization"`
	Auth         *AuthInfo         `json:"auth"`
}

// AuthInfo represents authentication information in whoami output
type AuthInfo struct {
	Method    string  `json:"method"`
	Source    string  `json:"source"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

// NewWhoamiCmd creates the whoami command
func NewWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Display current user and authentication status",
		Long: `Display information about the currently authenticated user.

Shows:
  - User details (name, email, admin status)
  - Organization/workspace information
  - Authentication method and source

Examples:
  linear whoami
  linear whoami --human`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Get auth status first
			authManager := auth.NewManager()
			authStatus, err := authManager.GetStatus(ctx)
			if err != nil {
				return fmt.Errorf("failed to get auth status: %w", err)
			}

			if !authStatus.Authenticated {
				if IsHumanOutput() {
					color.Red("Not authenticated")
					fmt.Println()
					fmt.Println("Run 'linear auth login' to authenticate")
					fmt.Println("Or set LINEAR_API_KEY environment variable")
				} else {
					OutputJSON(map[string]interface{}{
						"authenticated": false,
						"error":         "not authenticated",
					})
				}
				return nil
			}

			// Create API client and fetch viewer info
			client, err := api.NewClient(ctx)
			if err != nil {
				return fmt.Errorf("failed to create API client: %w", err)
			}

			viewer, err := client.GetViewer(ctx)
			if err != nil {
				return fmt.Errorf("failed to fetch user info: %w", err)
			}

			// Build response
			response := &WhoamiResponse{
				User:         &viewer.Viewer,
				Organization: &viewer.Organization,
				Auth: &AuthInfo{
					Method: string(authStatus.Method),
					Source: authStatus.Source,
				},
			}

			if authStatus.ExpiresAt != nil {
				expStr := authStatus.ExpiresAt.Format("2006-01-02T15:04:05Z")
				response.Auth.ExpiresAt = &expStr
			}

			if IsHumanOutput() {
				printHumanWhoami(response)
			} else {
				OutputJSON(response)
			}

			return nil
		},
	}
}

func printHumanWhoami(r *WhoamiResponse) {
	// User section
	color.Cyan("User")
	fmt.Printf("  Name:  %s\n", r.User.DisplayName)
	fmt.Printf("  Email: %s\n", r.User.Email)
	if r.User.Admin {
		fmt.Printf("  Role:  %s\n", color.YellowString("Admin"))
	} else {
		fmt.Printf("  Role:  Member\n")
	}
	fmt.Println()

	// Organization section
	color.Cyan("Organization")
	fmt.Printf("  Name: %s\n", r.Organization.Name)
	fmt.Printf("  URL:  https://linear.app/%s\n", r.Organization.UrlKey)
	fmt.Println()

	// Auth section
	color.Cyan("Authentication")
	fmt.Printf("  Method: %s\n", r.Auth.Method)
	fmt.Printf("  Source: %s\n", r.Auth.Source)
	if r.Auth.ExpiresAt != nil {
		fmt.Printf("  Expires: %s\n", *r.Auth.ExpiresAt)
	}
}

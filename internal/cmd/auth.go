package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/juanbermudez/agent-linear-cli/internal/api"
	"github.com/juanbermudez/agent-linear-cli/internal/auth"
	"github.com/juanbermudez/agent-linear-cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// NewAuthCmd creates the auth command group
func NewAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
		Long: `Authenticate with Linear using API keys or OAuth client credentials.

Authentication methods (in priority order):
  1. Environment variables: LINEAR_API_KEY or LINEAR_CLIENT_ID + LINEAR_CLIENT_SECRET
  2. System keychain (secure storage)
  3. Config file (legacy fallback)

Examples:
  linear auth                    # Interactive login (prompts for method)
  linear auth status             # Check authentication status
  linear auth logout             # Remove stored credentials`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Running "linear auth" without subcommand triggers interactive login
			return runInteractiveAuth()
		},
	}

	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthStatusCmd())
	cmd.AddCommand(newAuthLogoutCmd())
	cmd.AddCommand(newAuthTokenCmd())

	return cmd
}

// runInteractiveAuth prompts the user to choose an auth method
func runInteractiveAuth() error {
	manager := auth.NewManager()
	ctx := context.Background()

	fmt.Println("Linear CLI Authentication")
	fmt.Println()
	fmt.Println("Choose authentication method:")
	fmt.Println()
	fmt.Println("  " + color.CyanString("1") + ") API Key (personal use)")
	fmt.Println("     Get from: https://linear.app/settings/api")
	fmt.Println()
	fmt.Println("  " + color.CyanString("2") + ") Client Credentials (for AI agents/automation)")
	fmt.Println("     Create OAuth app at: https://linear.app/settings/api")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter choice [1/2]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1", "":
		// Default to API key
		fmt.Println()
		return loginWithAPIKey(manager, false, false)
	case "2":
		fmt.Println()
		return loginWithClientCredentials(ctx, manager, false)
	default:
		return fmt.Errorf("invalid choice: %s (enter 1 or 2)", choice)
	}
}

func newAuthLoginCmd() *cobra.Command {
	var (
		withToken         bool
		clientCredentials bool
		stdin             bool
		teamKey           string
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Linear",
		Long: `Authenticate with Linear using an API key or client credentials.

API Key (personal use):
  Get your API key from: https://linear.app/settings/api

Client Credentials (for agents/automation):
  Create an OAuth app at: https://linear.app/settings/api
  Enable "Client credentials" grant type

Examples:
  linear auth login                           # Interactive prompt
  linear auth login --with-token              # Paste API key
  linear auth login --with-token --team ENG   # Set up with default team
  linear auth login --client-credentials      # Set up OAuth client credentials
  echo $TOKEN | linear auth login --stdin     # Read from stdin (for scripts)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := auth.NewManager()
			ctx := context.Background()

			var err error
			if clientCredentials {
				err = loginWithClientCredentials(ctx, manager, stdin)
			} else {
				err = loginWithAPIKey(manager, withToken, stdin)
			}

			if err != nil {
				return err
			}

			// After successful auth, handle team setup
			return handlePostAuthTeamSetup(ctx, teamKey)
		},
	}

	cmd.Flags().BoolVar(&withToken, "with-token", false, "Read API key from prompt or stdin")
	cmd.Flags().BoolVar(&clientCredentials, "client-credentials", false, "Set up OAuth client credentials")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read credentials from stdin (non-interactive)")
	cmd.Flags().StringVar(&teamKey, "team", "", "Set default team key (e.g., ENG)")

	return cmd
}

func loginWithAPIKey(manager *auth.Manager, withToken, stdin bool) error {
	var apiKey string

	if stdin {
		// Read from stdin
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			apiKey = strings.TrimSpace(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
	} else if withToken {
		// Prompt for token (hidden input)
		fmt.Print("Paste your Linear API key: ")
		keyBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read API key: %w", err)
		}
		fmt.Println() // newline after hidden input
		apiKey = strings.TrimSpace(string(keyBytes))
	} else {
		// Interactive mode - show instructions
		fmt.Println("To authenticate, you need a Linear API key.")
		fmt.Println()
		fmt.Println("Get your API key from: " + color.CyanString("https://linear.app/settings/api"))
		fmt.Println()
		fmt.Print("Paste your API key: ")
		keyBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read API key: %w", err)
		}
		fmt.Println()
		apiKey = strings.TrimSpace(string(keyBytes))
	}

	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	if err := manager.LoginWithAPIKey(apiKey); err != nil {
		return err
	}

	if IsHumanOutput() {
		color.Green("✓ Authentication successful")
		fmt.Println("  Token stored securely in system keychain")
	} else {
		OutputJSON(map[string]interface{}{
			"success": true,
			"method":  "api_key",
			"storage": "keychain",
		})
	}

	return nil
}

func loginWithClientCredentials(ctx context.Context, manager *auth.Manager, stdin bool) error {
	var clientID, clientSecret string

	if stdin {
		// Read client_id and client_secret from stdin (one per line)
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			clientID = strings.TrimSpace(scanner.Text())
		}
		if scanner.Scan() {
			clientSecret = strings.TrimSpace(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
	} else {
		// Interactive mode
		fmt.Println("Setting up OAuth client credentials for agent authentication.")
		fmt.Println()
		fmt.Println("Create an OAuth app at: " + color.CyanString("https://linear.app/settings/api"))
		fmt.Println("Enable " + color.YellowString("Client credentials") + " grant type.")
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)

		fmt.Printf("Client ID [%s]: ", auth.DefaultClientID)
		input, _ := reader.ReadString('\n')
		clientID = strings.TrimSpace(input)
		if clientID == "" {
			clientID = auth.DefaultClientID
		}

		fmt.Print("Client Secret: ")
		secretBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read client secret: %w", err)
		}
		fmt.Println()
		clientSecret = strings.TrimSpace(string(secretBytes))
	}

	if clientSecret == "" {
		return fmt.Errorf("client secret cannot be empty")
	}

	if err := manager.LoginWithClientCredentials(ctx, clientID, clientSecret); err != nil {
		return err
	}

	if IsHumanOutput() {
		color.Green("✓ Authentication successful")
		fmt.Println("  Credentials stored securely in system keychain")
		fmt.Println("  Token will auto-refresh every 30 days")
	} else {
		OutputJSON(map[string]interface{}{
			"success": true,
			"method":  "client_credentials",
			"storage": "keychain",
		})
	}

	return nil
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		Long: `Display current authentication status and method.

Shows:
  - Whether you're authenticated
  - Authentication method (API key or client credentials)
  - Token source (environment, keychain, or config file)
  - Token expiry (for OAuth tokens)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := auth.NewManager()
			ctx := context.Background()

			status, err := manager.GetStatus(ctx)
			if err != nil {
				return err
			}

			if IsHumanOutput() {
				if status.Authenticated {
					color.Green("✓ Authenticated")
					fmt.Printf("  Method: %s\n", status.Method)
					fmt.Printf("  Source: %s\n", status.Source)
					if status.ExpiresAt != nil {
						fmt.Printf("  Expires: %s\n", status.ExpiresAt.Format("2006-01-02 15:04:05"))
					}
				} else {
					color.Red("✗ Not authenticated")
					fmt.Println()
					fmt.Println("Run 'linear auth' to authenticate")
					fmt.Println("Or set LINEAR_API_KEY environment variable")
				}
			} else {
				OutputJSON(status)
			}

			return nil
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials",
		Long: `Remove all stored credentials from the system keychain.

Note: This does not affect environment variables.
To fully logout, also unset LINEAR_API_KEY, LINEAR_CLIENT_ID, and LINEAR_CLIENT_SECRET.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := auth.NewManager()

			if err := manager.Logout(); err != nil {
				return err
			}

			if IsHumanOutput() {
				color.Green("✓ Logged out")
				fmt.Println("  Credentials removed from keychain")

				// Warn about environment variables
				if os.Getenv("LINEAR_API_KEY") != "" {
					color.Yellow("  Note: LINEAR_API_KEY is still set in environment")
				}
				if os.Getenv("LINEAR_CLIENT_ID") != "" {
					color.Yellow("  Note: LINEAR_CLIENT_ID is still set in environment")
				}
			} else {
				OutputJSON(map[string]interface{}{
					"success": true,
					"message": "credentials removed from keychain",
				})
			}

			return nil
		},
	}
}

func newAuthTokenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "token",
		Short: "Print current access token",
		Long: `Print the current access token to stdout.

Useful for piping to other commands or debugging.
The token is printed without a trailing newline.

Example:
  curl -H "Authorization: $(linear auth token)" https://api.linear.app/graphql`,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := auth.NewManager()
			ctx := context.Background()

			token, _, err := manager.GetToken(ctx)
			if err != nil {
				return err
			}

			// Print token without newline for piping
			fmt.Print(token)
			return nil
		},
	}
}

// handlePostAuthTeamSetup sets up team config after successful authentication
func handlePostAuthTeamSetup(ctx context.Context, teamKey string) error {
	// Create API client to fetch teams
	client, err := api.NewClient(ctx)
	if err != nil {
		// Auth succeeded but can't create client - just warn and continue
		if IsHumanOutput() {
			color.Yellow("  Warning: Could not verify team access")
		}
		return nil
	}

	// If team key provided, validate and set it
	if teamKey != "" {
		team, err := client.GetTeamByKey(ctx, teamKey)
		if err != nil || team == nil {
			return fmt.Errorf("team '%s' not found", teamKey)
		}

		manager, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("failed to save team config: %w", err)
		}

		if err := manager.Set("team_key", teamKey); err != nil {
			return fmt.Errorf("failed to save team key: %w", err)
		}
		if err := manager.Set("team_id", team.ID); err != nil {
			return fmt.Errorf("failed to save team id: %w", err)
		}

		if IsHumanOutput() {
			fmt.Printf("  Default team: %s (%s)\n", team.Key, team.Name)
		}
		return nil
	}

	// No team specified - show available teams
	teamsResp, err := client.GetTeams(ctx)
	if err != nil || teamsResp == nil || len(teamsResp.Teams) == 0 {
		return nil
	}

	if IsHumanOutput() {
		fmt.Println()
		fmt.Println("Available teams:")
		for _, t := range teamsResp.Teams {
			fmt.Printf("  %s - %s\n", t.Key, t.Name)
		}
		fmt.Println()
		fmt.Println("Set a default team with:")
		fmt.Println("  linear config set team_key <TEAM_KEY>")
		fmt.Println("Or re-run with --team flag:")
		fmt.Printf("  linear auth login --team %s\n", teamsResp.Teams[0].Key)
	}

	return nil
}

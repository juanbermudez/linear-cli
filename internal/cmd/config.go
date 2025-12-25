package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/juanbermudez/agent-linear-cli/internal/api"
	"github.com/juanbermudez/agent-linear-cli/internal/auth"
	"github.com/juanbermudez/agent-linear-cli/internal/config"
	"github.com/juanbermudez/agent-linear-cli/internal/output"
	"github.com/spf13/cobra"
)

// Valid configuration keys
var validConfigKeys = []string{
	"api_key",
	"team_id",
	"team_key",
}

// NewConfigCmd creates the config command group
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long: `View and modify CLI configuration settings.

Configuration is stored in ~/.linear.toml or ./.linear.toml

Available keys:
  api_key   - Linear API key (prefer using keychain via 'linear auth')
  team_id   - Default team ID
  team_key  - Default team key (e.g., ENG)

Examples:
  linear config list
  linear config get team_key
  linear config set team_key ENG`,
	}

	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigListCmd())
	cmd.AddCommand(newConfigPathCmd())
	cmd.AddCommand(newConfigSetupCmd())

	return cmd
}

func newConfigGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long: `Get a configuration value by key.

Available keys:
  api_key   - Linear API key
  team_id   - Default team ID
  team_key  - Default team key

Examples:
  linear config get team_key
  linear config get api_key`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			manager, err := config.NewManager()
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("CONFIG_ERROR", err.Error())
			}

			value, err := manager.Get(key)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("CONFIG_ERROR", err.Error())
			}

			if IsHumanOutput() {
				if value == "" {
					output.HumanLn("%s: %s", key, output.Muted("(not set)"))
				} else if key == "api_key" {
					// Mask API key for security
					masked := maskSecret(value)
					output.HumanLn("%s: %s", key, masked)
				} else {
					output.HumanLn("%s: %s", key, value)
				}
			} else {
				output.JSON(map[string]interface{}{
					"key":   key,
					"value": value,
				})
			}

			return nil
		},
	}

	return cmd
}

func newConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value.

Available keys:
  api_key   - Linear API key (prefer using 'linear auth' instead)
  team_id   - Default team ID
  team_key  - Default team key (e.g., ENG)

Examples:
  linear config set team_key ENG
  linear config set team_id abc123`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			// Validate key
			if !isValidConfigKey(key) {
				if IsHumanOutput() {
					output.ErrorHuman(fmt.Sprintf("Unknown config key: %s\nValid keys: %s", key, strings.Join(validConfigKeys, ", ")))
					return nil
				}
				return output.Error("INVALID_KEY", fmt.Sprintf("Unknown config key: %s", key))
			}

			manager, err := config.NewManager()
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("CONFIG_ERROR", err.Error())
			}

			if err := manager.Set(key, value); err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("CONFIG_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.SuccessHuman(fmt.Sprintf("Set %s", key))
				output.HumanLn("  Config file: %s", manager.Path())
			} else {
				output.JSON(map[string]interface{}{
					"success": true,
					"key":     key,
					"path":    manager.Path(),
				})
			}

			return nil
		},
	}

	return cmd
}

func newConfigListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		Long: `List all configuration values.

Shows values from:
  - Config file (~/.linear.toml or ./.linear.toml)
  - Environment variables (LINEAR_API_KEY, etc.)

Examples:
  linear config list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := config.NewManager()
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("CONFIG_ERROR", err.Error())
			}

			cfg, err := manager.Load()
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("CONFIG_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.HumanLn("Configuration (%s):\n", manager.Path())

				// API Key
				apiKeyValue := cfg.APIKey
				apiKeySource := "config"
				if envKey := os.Getenv("LINEAR_API_KEY"); envKey != "" {
					apiKeyValue = envKey
					apiKeySource = "env"
				}
				if apiKeyValue != "" {
					output.HumanLn("  api_key:  %s (%s)", maskSecret(apiKeyValue), apiKeySource)
				} else {
					output.HumanLn("  api_key:  %s", output.Muted("(not set)"))
				}

				// Team ID
				if cfg.TeamID != "" {
					output.HumanLn("  team_id:  %s", cfg.TeamID)
				} else {
					output.HumanLn("  team_id:  %s", output.Muted("(not set)"))
				}

				// Team Key
				if cfg.TeamKey != "" {
					output.HumanLn("  team_key: %s", cfg.TeamKey)
				} else {
					output.HumanLn("  team_key: %s", output.Muted("(not set)"))
				}

				// Environment variable hints
				output.HumanLn("")
				output.HumanLn("Environment variables:")
				printEnvVar("LINEAR_API_KEY")
				printEnvVar("LINEAR_CLIENT_ID")
				printEnvVar("LINEAR_CLIENT_SECRET")
				printEnvVar("LINEAR_TEAM")
			} else {
				configMap := map[string]interface{}{
					"api_key":  cfg.APIKey,
					"team_id":  cfg.TeamID,
					"team_key": cfg.TeamKey,
				}

				envVars := map[string]string{}
				for _, key := range []string{"LINEAR_API_KEY", "LINEAR_CLIENT_ID", "LINEAR_CLIENT_SECRET", "LINEAR_TEAM"} {
					if val := os.Getenv(key); val != "" {
						if strings.Contains(key, "KEY") || strings.Contains(key, "SECRET") {
							envVars[key] = "(set)"
						} else {
							envVars[key] = val
						}
					}
				}

				output.JSON(map[string]interface{}{
					"path":   manager.Path(),
					"config": configMap,
					"env":    envVars,
				})
			}

			return nil
		},
	}

	return cmd
}

func newConfigPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Show configuration file path",
		Long: `Show the path to the configuration file.

Examples:
  linear config path`,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := config.NewManager()
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(err.Error())
					return nil
				}
				return output.Error("CONFIG_ERROR", err.Error())
			}

			if IsHumanOutput() {
				output.HumanLn("%s", manager.Path())
			} else {
				output.JSON(map[string]interface{}{
					"path": manager.Path(),
				})
			}

			return nil
		},
	}

	return cmd
}

func newConfigSetupCmd() *cobra.Command {
	var (
		apiKey   string
		teamKey  string
		validate bool
		stdin    bool
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Set up CLI configuration",
		Long: `Set up CLI configuration with validation.

This command configures the CLI with your Linear API key and default team.
It validates the API key by making a test API call.

Examples:
  linear config setup --api-key lin_api_xxx --team ENG
  linear config setup --api-key lin_api_xxx
  linear config setup --validate
  echo "lin_api_xxx" | linear config setup --stdin --team ENG`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// If validate flag is set, just validate existing config
			if validate {
				return validateConfig(ctx)
			}

			// Read API key from stdin if requested
			if stdin {
				reader := bufio.NewReader(os.Stdin)
				line, err := reader.ReadString('\n')
				if err != nil && line == "" {
					if IsHumanOutput() {
						output.ErrorHuman("Failed to read API key from stdin")
						return nil
					}
					return output.Error("STDIN_ERROR", "Failed to read API key from stdin")
				}
				apiKey = strings.TrimSpace(line)
			}

			// Require API key if not validating
			if apiKey == "" {
				if IsHumanOutput() {
					output.ErrorHuman("API key is required. Use --api-key or --stdin flag.")
					output.HumanLn("\nTo get an API key:")
					output.HumanLn("  1. Go to: https://linear.app/settings/api")
					output.HumanLn("  2. Create a new Personal API key")
					output.HumanLn("  3. Run: linear config setup --api-key <your-key>")
					output.HumanLn("     Or:  echo <your-key> | linear config setup --stdin")
					return nil
				}
				return output.ErrorWithHint("MISSING_API_KEY", "API key is required",
					"Get your API key from https://linear.app/settings/api",
					"linear config setup --api-key lin_api_xxx",
					"echo $LINEAR_API_KEY | linear config setup --stdin")
			}

			// Validate API key format
			if !strings.HasPrefix(apiKey, "lin_api_") {
				if IsHumanOutput() {
					output.ErrorHuman("Invalid API key format. Must start with 'lin_api_'")
					return nil
				}
				return output.Error("INVALID_API_KEY", "API key must start with 'lin_api_'")
			}

			// Store API key in keychain
			authManager := auth.NewManager()
			if err := authManager.LoginWithAPIKey(apiKey); err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(fmt.Sprintf("Failed to store API key: %s", err.Error()))
					return nil
				}
				return output.Error("STORE_ERROR", err.Error())
			}

			// Validate by making a test API call
			client, err := api.NewClient(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(fmt.Sprintf("Failed to create API client: %s", err.Error()))
					return nil
				}
				return output.Error("API_ERROR", err.Error())
			}

			viewer, err := client.GetViewer(ctx)
			if err != nil {
				if IsHumanOutput() {
					output.ErrorHuman(fmt.Sprintf("API key validation failed: %s", err.Error()))
					return nil
				}
				return output.Error("VALIDATION_ERROR", err.Error())
			}

			// Set team key if provided
			if teamKey != "" {
				manager, err := config.NewManager()
				if err != nil {
					if IsHumanOutput() {
						output.ErrorHuman(err.Error())
						return nil
					}
					return output.Error("CONFIG_ERROR", err.Error())
				}

				// Validate team exists
				team, err := client.GetTeamByKey(ctx, teamKey)
				if err != nil || team == nil {
					if IsHumanOutput() {
						output.ErrorHuman(fmt.Sprintf("Team '%s' not found", teamKey))
						return nil
					}
					return output.Error("TEAM_NOT_FOUND", fmt.Sprintf("Team '%s' not found", teamKey))
				}

				if err := manager.Set("team_key", teamKey); err != nil {
					if IsHumanOutput() {
						output.ErrorHuman(err.Error())
						return nil
					}
					return output.Error("CONFIG_ERROR", err.Error())
				}

				if err := manager.Set("team_id", team.ID); err != nil {
					if IsHumanOutput() {
						output.ErrorHuman(err.Error())
						return nil
					}
					return output.Error("CONFIG_ERROR", err.Error())
				}
			}

			// Get available teams to show to user
			var availableTeams []api.Team
			if teamKey == "" {
				teamsResp, err := client.GetTeams(ctx)
				if err == nil && teamsResp != nil {
					availableTeams = teamsResp.Teams
				}
			}

			if IsHumanOutput() {
				output.SuccessHuman("Configuration complete")
				output.HumanLn("")
				output.HumanLn("Authenticated as: %s (%s)", viewer.Viewer.DisplayName, viewer.Viewer.Email)
				if teamKey != "" {
					output.HumanLn("Default team: %s", teamKey)
				} else if len(availableTeams) > 0 {
					output.HumanLn("")
					output.HumanLn("Available teams:")
					for _, t := range availableTeams {
						output.HumanLn("  %s - %s", t.Key, t.Name)
					}
					output.HumanLn("")
					output.HumanLn("Set a default team with:")
					output.HumanLn("  linear config set team_key <TEAM_KEY>")
				}
				output.HumanLn("")
				output.HumanLn("Run 'linear whoami' to verify your configuration")
			} else {
				result := map[string]interface{}{
					"success": true,
					"user": map[string]string{
						"id":          viewer.Viewer.ID,
						"displayName": viewer.Viewer.DisplayName,
						"email":       viewer.Viewer.Email,
					},
				}
				if teamKey != "" {
					result["team"] = teamKey
				}
				if len(availableTeams) > 0 {
					teamList := make([]map[string]string, len(availableTeams))
					for i, t := range availableTeams {
						teamList[i] = map[string]string{
							"id":   t.ID,
							"key":  t.Key,
							"name": t.Name,
						}
					}
					result["availableTeams"] = teamList
				}
				output.JSON(result)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&apiKey, "api-key", "", "Linear API key (lin_api_...)")
	cmd.Flags().StringVar(&teamKey, "team", "", "Default team key (e.g., ENG)")
	cmd.Flags().BoolVar(&validate, "validate", false, "Validate existing configuration")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read API key from stdin")

	return cmd
}

func validateConfig(ctx context.Context) error {
	client, err := api.NewClient(ctx)
	if err != nil {
		if IsHumanOutput() {
			output.ErrorHuman(fmt.Sprintf("Configuration invalid: %s", err.Error()))
			return nil
		}
		return output.Error("INVALID_CONFIG", err.Error())
	}

	viewer, err := client.GetViewer(ctx)
	if err != nil {
		if IsHumanOutput() {
			output.ErrorHuman(fmt.Sprintf("API validation failed: %s", err.Error()))
			return nil
		}
		return output.Error("VALIDATION_ERROR", err.Error())
	}

	if IsHumanOutput() {
		output.SuccessHuman("Configuration is valid")
		output.HumanLn("  Authenticated as: %s (%s)", viewer.Viewer.DisplayName, viewer.Viewer.Email)
	} else {
		output.JSON(map[string]interface{}{
			"valid": true,
			"user": map[string]string{
				"id":          viewer.Viewer.ID,
				"displayName": viewer.Viewer.DisplayName,
				"email":       viewer.Viewer.Email,
			},
		})
	}

	return nil
}

// Helper functions

func isValidConfigKey(key string) bool {
	for _, valid := range validConfigKeys {
		if key == valid {
			return true
		}
	}
	return false
}

func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + "..." + secret[len(secret)-4:]
}

func printEnvVar(name string) {
	value := os.Getenv(name)
	if value != "" {
		if strings.Contains(name, "KEY") || strings.Contains(name, "SECRET") {
			color.Green("  %s: (set)", name)
		} else {
			color.Green("  %s: %s", name, value)
		}
	} else {
		output.HumanLn("  %s: %s", name, output.Muted("(not set)"))
	}
}

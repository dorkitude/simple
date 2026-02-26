package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/dorkitude/simple/internal/client"
	"github.com/dorkitude/simple/internal/config"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with DNSimple",
	Long:  `Manage authentication with the DNSimple API.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login with a DNSimple API token",
	Long:  `Prompts for your API token and validates it against the DNSimple API.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		if config.HasToken() {
			fmt.Println(ui.Warn("Already authenticated. Use '" + BinName() + " auth logout' first to re-authenticate."))
			return nil
		}

		fmt.Println(ui.TitleStyle.Render("🔐 DNSimple Authentication"))
		fmt.Println()
		fmt.Print("Enter your API token: ")

		reader := bufio.NewReader(os.Stdin)
		token, _ := reader.ReadString('\n')
		token = strings.TrimSpace(token)

		if token == "" {
			return fmt.Errorf("token cannot be empty")
		}

		fmt.Println()
		fmt.Println(ui.SubtleStyle.Render("Validating token..."))

		whoami, err := client.ValidateToken(ctx, token, sandboxFlag)
		if err != nil {
			return err
		}

		// Save token
		if err := config.SaveToken(token); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}

		// Resolve and cache account ID
		cfg := &config.Config{Sandbox: sandboxFlag}
		if whoami.Account != nil {
			cfg.AccountID = strconv.FormatInt(whoami.Account.ID, 10)
		} else if whoami.User != nil {
			// User token — try to auto-resolve account
			app, err := client.NewFromFlags(ctx, "", sandboxFlag)
			if err == nil {
				cfg.AccountID = app.AccountID
			}
		}
		_ = config.Save(cfg)

		fmt.Println()
		fmt.Println(ui.Success("Authenticated with DNSimple! 🎉"))
		if whoami.Account != nil {
			fmt.Println(ui.SubtleStyle.Render(fmt.Sprintf("Account: %s (ID: %d)", whoami.Account.Email, whoami.Account.ID)))
		} else if whoami.User != nil {
			fmt.Println(ui.SubtleStyle.Render(fmt.Sprintf("User: %s (ID: %d)", whoami.User.Email, whoami.User.ID)))
		}

		tokenPath, _ := config.TokenPath()
		fmt.Println(ui.SubtleStyle.Render("Token saved to: " + tokenPath))

		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials",
	Long:  `Removes the locally stored API token.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.RemoveToken(); err != nil {
			return fmt.Errorf("failed to logout: %w", err)
		}
		fmt.Println(ui.Success("Logged out."))
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	Long:  `Check if you are currently authenticated with DNSimple.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if config.HasToken() {
			fmt.Println(ui.Success("Authenticated"))
			tokenPath, _ := config.TokenPath()
			fmt.Println(ui.SubtleStyle.Render("Token location: " + tokenPath))

			cfg, _ := config.Load()
			if cfg != nil && cfg.AccountID != "" {
				fmt.Println(ui.SubtleStyle.Render("Account ID: " + cfg.AccountID))
			}
		} else {
			fmt.Println(ui.Warn("Not authenticated"))
			fmt.Println(ui.SubtleStyle.Render("Run '" + BinName() + " auth login' to authenticate"))
		}
		return nil
	},
}

var authSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Instructions for getting a DNSimple API token",
	Long:  `Prints instructions for creating a DNSimple API token.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(ui.TitleStyle.Render("🔧 DNSimple API Token Setup"))
		fmt.Println()
		fmt.Println(ui.SubtleStyle.Render("DNSimple uses API tokens for authentication. Here's how to get one:"))
		fmt.Println()
		fmt.Println(ui.SuccessStyle.Render("1.") + " Log in to your DNSimple account at https://dnsimple.com")
		fmt.Println()
		fmt.Println(ui.SuccessStyle.Render("2.") + " Go to Account → Access Tokens")
		fmt.Println("   → https://dnsimple.com/a/YOUR_ACCOUNT_ID/account/access_tokens")
		fmt.Println()
		fmt.Println(ui.SuccessStyle.Render("3.") + " Click " + ui.AccentStyle.Render("\"New access token\""))
		fmt.Println("   → Give it a name (e.g., 'simple')")
		fmt.Println("   → Copy the generated token")
		fmt.Println()
		fmt.Println(ui.SuccessStyle.Render("4.") + " Run: " + ui.AccentStyle.Render(BinName()+" auth login"))
		fmt.Println("   → Paste your token when prompted")
		fmt.Println()
		fmt.Println(ui.SubtleStyle.Render("For sandbox testing, use https://sandbox.dnsimple.com"))
		fmt.Println(ui.SubtleStyle.Render("and pass --sandbox when running commands."))
		fmt.Println()
		fmt.Println(ui.SubtleStyle.Render("That's it! Much simpler than OAuth. 🎉"))
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authSetupCmd)
}

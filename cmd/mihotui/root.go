package main

import (
	"fmt"
	"os"

	"github.com/jackie8tao/mihotui/internal/app"
	"github.com/jackie8tao/mihotui/internal/config"
	"github.com/jackie8tao/mihotui/internal/profile"
	"github.com/jackie8tao/mihotui/pkg/mihomo"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	configPath string
)

var rootCmd = &cobra.Command{
	Use:   "mihotui",
	Short: "Modern TUI Manager for Mihomo (Clash.Meta)",
	Long: `Mihomo TUI (mihotui) is a lightweight, interactive terminal dashboard
and management tool for Mihomo (Clash.Meta) on headless Linux servers.

Run without any subcommands to launch the interactive TUI interface.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI(configPath)
	},
}

func init() {
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file")
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built at: %s)", version, commit, buildDate)
	rootCmd.SetVersionTemplate("mihotui {{.Version}}\n")
}

func runTUI(cfgPath string) error {
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	appPaths, err := config.ResolveAppPaths()
	if err != nil {
		return fmt.Errorf("failed to resolve app paths: %w", err)
	}

	profileMgr, err := profile.NewManager(appPaths.MihomoDir,
		profile.WithActiveProfile(cfg.ActiveProfile),
		profile.WithSubscriptions(cfg.Subscriptions),
		profile.WithSaveFunc(func(activeProfile string, subs []profile.Subscription) error {
			cfg.ActiveProfile = activeProfile
			cfg.Subscriptions = subs
			return config.SaveConfig(cfg)
		}),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize profile manager: %v\n", err)
	}

	mihoCli, err := mihomo.New(cfg.ApiURL,
		mihomo.WithSecret(cfg.Secret),
	)
	if err != nil {
		return fmt.Errorf("failed to create Mihomo API client: %w", err)
	}

	appModel := app.New(mihoCli, cfg, profileMgr)
	p := tea.NewProgram(
		appModel,
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI application: %w", err)
	}

	return nil
}


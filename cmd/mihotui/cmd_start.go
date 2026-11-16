package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/jackie8tao/mihotui/internal/app"
	"github.com/jackie8tao/mihotui/internal/config"
	"github.com/jackie8tao/mihotui/internal/profile"
	"github.com/jackie8tao/mihotui/pkg/mihomo"
)

var (
	configPath string
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Launch the interactive TUI interface",
	Long:  "Launch the interactive terminal user interface (TUI) dashboard to monitor and manage Mihomo.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI(configPath)
	},
}

func init() {
	startCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file")
	rootCmd.AddCommand(startCmd)
}

func runTUI(cfgPath string) error {
	appPaths, err := config.ResolveAppPaths()
	if err != nil {
		return fmt.Errorf("failed to resolve app paths: %w", err)
	}

	cfg, err := config.LoadConfig(cfgPath, appPaths.MihotuiConfig)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	profileMgr, err := profile.NewManager(appPaths.MihomoDir,
		profile.WithActiveProfile(cfg.ActiveProfile),
		profile.WithSubscriptions(cfg.Subscriptions),
		profile.WithSaveFunc(func(activeProfile string, subs []config.Subscription) error {
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

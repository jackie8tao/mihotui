package main

import (
	"fmt"
	"path/filepath"

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

// runTUI resolves suite paths, loads configuration, initializes dependencies, and runs the Bubble Tea program.
func runTUI(cfgPath string) error {
	appPaths, err := config.ResolveAppPaths()
	if err != nil {
		return err
	}

	if cfgPath == "" {
		cfgPath = filepath.Join(appPaths.AppEtcDir(), "config.yaml")
	}
	appCfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return err
	}

	profileMgr, err := profile.NewManager(appCfg, appPaths)
	if err != nil {
		return err
	}

	mihoCli, err := mihomo.New(appCfg.ApiURL,
		mihomo.WithSecret(appCfg.Secret),
	)
	if err != nil {
		return err
	}

	appModel := app.New(mihoCli, profileMgr)
	p := tea.NewProgram(
		appModel,
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run TUI: %w", err)
	}

	return nil
}

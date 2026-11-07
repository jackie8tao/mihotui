package main

import (
	"flag"
	"fmt"
	"os"

	"mihomo-tui/internal/app"
	"mihomo-tui/internal/config"
	"mihomo-tui/internal/profile"
	"mihomo-tui/pkg/mihomo"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	version   = "v0.1.0"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	configPath := flag.String("c", "", "Path to configuration file")
	apiURL := flag.String("u", "", "Mihomo external controller API URL (e.g. http://127.0.0.1:9090)")
	secret := flag.String("s", "", "Mihomo API secret token")
	configDir := flag.String("d", "", "Mihomo configurations directory (e.g. /etc/mihomo)")
	showVersion := flag.Bool("v", false, "Show mihomo-tui version")

	flag.Parse()

	if *showVersion {
		fmt.Printf("mihomo-tui %s (commit: %s, built at: %s)\n", version, commit, buildDate)
		return
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if *apiURL != "" {
		cfg.ApiURL = *apiURL
	}
	if *secret != "" {
		cfg.Secret = *secret
	}
	if *configDir != "" {
		cfg.MihomoDir = *configDir
	}

	// Create Profile Manager
	profileMgr, err := profile.NewManager(cfg.MihomoDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize profile manager: %v\n", err)
	}

	// Create API client with Functional Options
	apiClient, err := mihomo.New(cfg.ApiURL,
		mihomo.WithSecret(cfg.Secret),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Mihomo client: %v\n", err)
		os.Exit(1)
	}

	// Create Bubble Tea Model
	appModel := app.New(apiClient, cfg, profileMgr)

	// Launch TUI in fullscreen alternative screen buffer
	p := tea.NewProgram(
		appModel,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}

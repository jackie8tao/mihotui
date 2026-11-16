package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mihotui",
	Short: "Modern TUI Manager for Mihomo (Clash.Meta)",
	Long: `Mihomo TUI (mihotui) is a lightweight, interactive terminal dashboard
and management tool for Mihomo (Clash.Meta) on headless Linux servers.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built at: %s)", version, commit, buildDate)
	rootCmd.SetVersionTemplate("mihotui {{.Version}}\n")
}

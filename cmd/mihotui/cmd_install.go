package main

import (
	"context"
	"fmt"

	"github.com/jackie8tao/mihotui/internal/config"
	"github.com/jackie8tao/mihotui/internal/installer"

	"github.com/spf13/cobra"
)

var (
	installTargetDir   string
	installArchivePath string
	installForce       bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Mihomo core binary, suite directories, and systemd service unit",
	Long: `Download and install the Mihomo (Clash.Meta) core binary into the self-contained
bundle directory, or extract directly from a local gzip archive (.gz),
initialize its configuration directory, and install the systemd unit file on Linux.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := installer.Options{
			ArchivePath: installArchivePath,
			Force:       installForce,
		}
		if installTargetDir != "" {
			opts.AppPaths = config.NewAppPaths(installTargetDir)
		} else {
			paths, err := config.ResolveAppPaths()
			if err != nil {
				return fmt.Errorf("resolve app paths: %w", err)
			}
			opts.AppPaths = paths
		}
		return installer.Install(context.Background(), opts)
	},
}

func init() {
	installCmd.Flags().StringVarP(&installTargetDir, "directory", "d", "", "Target bundle root directory (defaults to current suite root)")
	installCmd.Flags().StringVarP(&installArchivePath, "archive", "a", "", "Local Mihomo gzip archive (.gz) for offline installation")
	installCmd.Flags().BoolVarP(&installForce, "force", "f", false, "Force overwrite existing binary")

	rootCmd.AddCommand(installCmd)
}

package main

import (
	"context"
	"fmt"

	"mihomo-tui/internal/config"
	"mihomo-tui/internal/installer"

	"github.com/spf13/cobra"
)

var (
	installTargetDir string
	installCoreVer   string
	installMirror    string
	installForce     bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Mihomo core binary and initialize bundle directory",
	Long: `Download and install the Mihomo (Clash.Meta) core binary into the self-contained
bundle directory (by default relative to mihotui or /usr/local/mihotui), and initialize
its configuration directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := installTargetDir
		if targetDir == "" {
			paths := config.ResolveAppPaths()
			if paths.IsBundle {
				targetDir = paths.AppRoot
			} else {
				targetDir = "/usr/local/mihotui"
			}
		}

		opts := installer.Options{
			TargetDir: targetDir,
			Version:   installCoreVer,
			Mirror:    installMirror,
			Force:     installForce,
		}

		res, err := installer.Install(context.Background(), opts)
		if err != nil {
			return fmt.Errorf("installation failed: %w", err)
		}

		fmt.Println("\n==================================================")
		fmt.Printf("Mihomo Core successfully installed into bundle!\n")
		fmt.Printf("Bundle Root  : %s\n", res.TargetDir)
		fmt.Printf("Core Binary  : %s\n", res.MihomoBin)
		fmt.Printf("Core Config  : %s\n", res.MihomoConfig)
		fmt.Println("==================================================")
		return nil
	},
}

func init() {
	paths := config.ResolveAppPaths()
	defaultDir := paths.AppRoot
	if !paths.IsBundle {
		defaultDir = "/usr/local/mihotui"
	}

	installCmd.Flags().StringVarP(&installTargetDir, "dir", "d", defaultDir, "Target bundle root directory")
	installCmd.Flags().StringVarP(&installCoreVer, "version", "v", "latest", "Mihomo core release version to install")
	installCmd.Flags().StringVarP(&installMirror, "mirror", "m", "", "GitHub download mirror prefix (e.g. https://ghproxy.net/)")
	installCmd.Flags().BoolVarP(&installForce, "force", "f", false, "Force overwrite existing binary")

	rootCmd.AddCommand(installCmd)
}


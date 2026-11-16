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
	installCoreVer     string
	installMirror      string
	installForce       bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Mihomo core binary and initialize bundle directory",
	Long: `Download and install the Mihomo (Clash.Meta) core binary into the self-contained
bundle directory, or extract directly from a local compressed archive (.gz, .tar.gz, .tgz, .zip),
and initialize its configuration directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var paths *config.AppPaths
		var err error
		if installTargetDir != "" {
			paths = config.NewAppPaths(installTargetDir)
		} else {
			paths, err = config.ResolveAppPaths()
			if err != nil {
				return err
			}
		}

		opts := installer.Options{
			Paths:       paths,
			ArchivePath: installArchivePath,
			Version:     installCoreVer,
			Mirror:      installMirror,
			Force:       installForce,
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
	installCmd.Flags().StringVarP(&installTargetDir, "dir", "d", "", "Target bundle root directory (defaults to current suite root)")
	installCmd.Flags().StringVarP(&installArchivePath, "archive", "a", "", "Local Mihomo release archive (.gz, .tar.gz, .tgz, .zip) for offline installation")
	installCmd.Flags().StringVarP(&installCoreVer, "version", "v", "latest", "Mihomo core release version to install")
	installCmd.Flags().StringVarP(&installMirror, "mirror", "m", "", "GitHub download mirror prefix (e.g. https://ghproxy.net/)")
	installCmd.Flags().BoolVarP(&installForce, "force", "f", false, "Force overwrite existing binary")

	rootCmd.AddCommand(installCmd)
}

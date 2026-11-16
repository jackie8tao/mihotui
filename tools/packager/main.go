package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Target struct {
	OS     string
	Arch   string
	ArmVer string
	Suffix string
}

var allTargets = []Target{
	{OS: "linux", Arch: "amd64", Suffix: "linux-amd64"},
	{OS: "linux", Arch: "arm64", Suffix: "linux-arm64"},
	{OS: "linux", Arch: "arm", ArmVer: "7", Suffix: "linux-armv7"},
	{OS: "linux", Arch: "386", Suffix: "linux-386"},
}

func main() {
	var version string
	var arch string
	flag.StringVar(&version, "v", "v0.1.0", "Release version tag (e.g. v0.1.0)")
	flag.StringVar(&arch, "arch", "amd64", "Target architecture: amd64, arm64, armv7, 386, or all")
	flag.Parse()

	var selectedTargets []Target
	if arch == "all" {
		selectedTargets = allTargets
	} else {
		for _, t := range allTargets {
			if t.Arch == arch || (arch == "armv7" && t.Suffix == "linux-armv7") {
				selectedTargets = append(selectedTargets, t)
			}
		}
		if len(selectedTargets) == 0 {
			fmt.Printf("Unknown arch %q. Available: amd64, arm64, armv7, 386, all\n", arch)
			os.Exit(1)
		}
	}

	// Detect git commit hash
	commit := "none"
	if out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output(); err == nil {
		commit = strings.TrimSpace(string(out))
	}

	buildDate := time.Now().UTC().Format(time.RFC3339)

	distDir := "dist"
	if arch == "all" {
		if err := os.RemoveAll(distDir); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Error cleaning dist directory: %v\n", err)
			os.Exit(1)
		}
	}
	if err := os.MkdirAll(distDir, 0755); err != nil {
		fmt.Printf("Error creating dist directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("==================================================")
	fmt.Printf("Building Mihomo TUI Release Packages (%s)\n", arch)
	fmt.Printf("Version : %s\nCommit  : %s\nDate    : %s\n", version, commit, buildDate)
	fmt.Println("==================================================")

	var checksums []string

	for _, t := range selectedTargets {
		fmt.Printf("\n==> Building for %s/%s%s...\n", t.OS, t.Arch, func() string {
			if t.ArmVer != "" {
				return " (v" + t.ArmVer + ")"
			}
			return ""
		}())

		binName := "mihotui"
		tempBinPath := filepath.Join(distDir, fmt.Sprintf("%s-%s", binName, t.Suffix))

		ldflags := fmt.Sprintf("-s -w -X main.version=%s -X main.commit=%s -X main.buildDate=%s",
			version, commit, buildDate)

		cmd := exec.Command("go", "build", "-trimpath", "-ldflags", ldflags, "-o", tempBinPath, "./cmd/mihotui")
		cmd.Env = append(os.Environ(),
			"GOOS="+t.OS,
			"GOARCH="+t.Arch,
			"CGO_ENABLED=0",
		)
		if t.ArmVer != "" {
			cmd.Env = append(cmd.Env, "GOARM="+t.ArmVer)
		}

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to compile %s: %v\n", t.Suffix, err)
			os.Exit(1)
		}

		// Also provide convenient flat uncompressed binary in dist/
		if len(selectedTargets) == 1 {
			flatBinPath := filepath.Join(distDir, binName)
			if err := copyFile(tempBinPath, flatBinPath, 0755); err != nil {
				fmt.Printf("Failed to copy flat binary %s: %v\n", flatBinPath, err)
				os.Exit(1)
			}
		}

		// Package into tar.gz
		tarGzName := fmt.Sprintf("mihotui-%s-%s.tar.gz", version, t.Suffix)
		tarGzPath := filepath.Join(distDir, tarGzName)

		if err := createTarGz(tarGzPath, tempBinPath); err != nil {
			fmt.Printf("Failed to package %s: %v\n", tarGzName, err)
			os.Exit(1)
		}

		// Calculate SHA256 checksum
		hash, err := fileSHA256(tarGzPath)
		if err != nil {
			fmt.Printf("Failed to compute sha256 for %s: %v\n", tarGzName, err)
			os.Exit(1)
		}

		checksumLine := fmt.Sprintf("%s  %s", hash, tarGzName)
		checksums = append(checksums, checksumLine)
		fmt.Printf("Package created: %s (SHA256: %s)\n", tarGzName, hash)

		// Create legacy package name for backward compatibility
		legacyTarGzName := fmt.Sprintf("mihomo-tui-%s-%s.tar.gz", version, t.Suffix)
		legacyTarGzPath := filepath.Join(distDir, legacyTarGzName)
		if err := copyFile(tarGzPath, legacyTarGzPath, 0644); err != nil {
			fmt.Printf("Failed to create legacy package %s: %v\n", legacyTarGzName, err)
			os.Exit(1)
		}
		legacyHash, err := fileSHA256(legacyTarGzPath)
		if err != nil {
			fmt.Printf("Failed to compute sha256 for %s: %v\n", legacyTarGzName, err)
			os.Exit(1)
		}
		checksums = append(checksums, fmt.Sprintf("%s  %s", legacyHash, legacyTarGzName))
	}

	// Write or append to sha256sums.txt
	checksumFilePath := filepath.Join(distDir, "sha256sums.txt")
	if err := os.WriteFile(checksumFilePath, []byte(strings.Join(checksums, "\n")+"\n"), 0644); err != nil {
		fmt.Printf("Error writing checksums file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n==================================================")
	fmt.Printf("Package(s) successfully created in ./%s/\n", distDir)
	fmt.Println("==================================================")
	for _, l := range checksums {
		fmt.Println(" " + l)
	}
}

func createTarGz(tarGzPath, binPath string) error {
	tarFile, err := os.Create(tarGzPath)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	gw := gzip.NewWriter(tarFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	entries := []struct {
		src      string
		dst      string
		mode     int64
		optional bool
	}{
		// 1. Core binary (required)
		{src: binPath, dst: "mihotui", mode: 0755, optional: false},

		// 2. Configuration examples
		{src: filepath.Join("configs", "config.example.yaml"), dst: "configs/config.example.yaml", mode: 0644, optional: true},
		{src: filepath.Join("configs", "config.example.yaml"), dst: "config.example.yaml", mode: 0644, optional: true},
		{src: filepath.Join("configs", "subscriptions.example.yaml"), dst: "configs/subscriptions.example.yaml", mode: 0644, optional: true},

		// 3. Documentation & deployment scripts
		{src: "README.md", dst: "README.md", mode: 0644, optional: true},
		{src: filepath.Join("scripts", "install.sh"), dst: "install.sh", mode: 0755, optional: true},
		{src: filepath.Join("scripts", "systemd", "mihomo.service"), dst: "mihomo.service", mode: 0644, optional: true},
	}

	for _, e := range entries {
		if _, err := os.Stat(e.src); err != nil {
			if e.optional && os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("stat %s failed: %w", e.src, err)
		}
		if err := addFileToTar(tw, e.src, e.dst, e.mode); err != nil {
			return fmt.Errorf("failed to add %s to tar: %w", e.dst, err)
		}
	}

	return nil
}

func addFileToTar(tw *tar.Writer, srcPath, tarName string, mode int64) error {
	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}

	header.Name = filepath.ToSlash(tarName)
	header.Mode = mode

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	file, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(tw, file)
	return err
}

func fileSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

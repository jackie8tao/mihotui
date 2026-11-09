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

var targets = []Target{
	{OS: "linux", Arch: "amd64", Suffix: "linux-amd64"},
	{OS: "linux", Arch: "arm64", Suffix: "linux-arm64"},
	{OS: "linux", Arch: "arm", ArmVer: "7", Suffix: "linux-armv7"},
	{OS: "linux", Arch: "386", Suffix: "linux-386"},
}

func main() {
	var version string
	flag.StringVar(&version, "v", "v0.1.0", "Release version tag (e.g. v0.1.0)")
	flag.Parse()

	// Detect git commit hash
	commit := "none"
	if out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output(); err == nil {
		commit = strings.TrimSpace(string(out))
	}

	buildDate := time.Now().UTC().Format(time.RFC3339)

	distDir := "dist"
	_ = os.RemoveAll(distDir)
	if err := os.MkdirAll(distDir, 0755); err != nil {
		fmt.Printf("Error creating dist directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("==================================================")
	fmt.Printf("Building Mihomo TUI Linux Release Packages\n")
	fmt.Printf("Version : %s\nCommit  : %s\nDate    : %s\n", version, commit, buildDate)
	fmt.Println("==================================================")

	var checksums []string

	for _, t := range targets {
		fmt.Printf("\n==> Building for %s/%s%s...\n", t.OS, t.Arch, func() string {
			if t.ArmVer != "" {
				return " (v" + t.ArmVer + ")"
			}
			return ""
		}())

		binName := "mihomo-tui"
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

		// Package into tar.gz
		tarGzName := fmt.Sprintf("mihomo-tui-%s-%s.tar.gz", version, t.Suffix)
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
	}

	// Write sha256sums.txt
	checksumFilePath := filepath.Join(distDir, "sha256sums.txt")
	if err := os.WriteFile(checksumFilePath, []byte(strings.Join(checksums, "\n")+"\n"), 0644); err != nil {
		fmt.Printf("Error writing checksums file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n==================================================")
	fmt.Printf("All Linux packages successfully created in ./%s/\n", distDir)
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

	// 1. Add binary (with executable mode 0755)
	if err := addFileToTar(tw, binPath, "mihomo-tui", 0755); err != nil {
		return err
	}

	// 2. Add configs directory files
	configFiles := []string{"config.example.yaml", "subscriptions.example.yaml"}
	for _, cf := range configFiles {
		src := filepath.Join("configs", cf)
		if _, err := os.Stat(src); err == nil {
			_ = addFileToTar(tw, src, filepath.Join("configs", cf), 0644)
			// Also add config.example.yaml at root of archive for ease of access
			if cf == "config.example.yaml" {
				_ = addFileToTar(tw, src, "config.example.yaml", 0644)
			}
		}
	}

	// 3. Add README.md (if exists)
	if _, err := os.Stat("README.md"); err == nil {
		_ = addFileToTar(tw, "README.md", "README.md", 0644)
	}

	// 4. Add install.sh
	installScript := filepath.Join("scripts", "install.sh")
	if _, err := os.Stat(installScript); err == nil {
		_ = addFileToTar(tw, installScript, "install.sh", 0755)
	}

	// 5. Add mihomo.service
	serviceFile := filepath.Join("scripts", "systemd", "mihomo.service")
	if _, err := os.Stat(serviceFile); err == nil {
		_ = addFileToTar(tw, serviceFile, "mihomo.service", 0644)
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

	header.Name = tarName
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

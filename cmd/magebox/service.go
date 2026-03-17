package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"qoliber/magebox/internal/cli"
	"qoliber/magebox/internal/platform"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage MageBox autostart service",
	Long:  "Install or remove a system service that automatically starts MageBox on login",
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install autostart service",
	Long:  "Installs a system service that starts global services and all projects on login",
	RunE:  runServiceInstall,
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove autostart service",
	Long:  "Removes the MageBox autostart service",
	RunE:  runServiceUninstall,
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show autostart service status",
	Long:  "Shows whether the MageBox autostart service is installed and active",
	RunE:  runServiceStatus,
}

func init() {
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
	serviceCmd.AddCommand(serviceStatusCmd)
	rootCmd.AddCommand(serviceCmd)
}

const (
	systemdServiceName = "magebox"
	launchdLabel       = "com.qoliber.magebox"
)

// getMageboxBinary returns the path to the magebox binary
func getMageboxBinary() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not determine magebox binary path: %w", err)
	}
	// Resolve symlinks to get the real path
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", fmt.Errorf("could not resolve magebox binary path: %w", err)
	}
	return execPath, nil
}

// systemd paths
func systemdUserDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user")
}

func systemdServicePath() string {
	return filepath.Join(systemdUserDir(), systemdServiceName+".service")
}

// launchd paths
func launchdAgentDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents")
}

func launchdPlistPath() string {
	return filepath.Join(launchdAgentDir(), launchdLabel+".plist")
}

func runServiceInstall(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	mageboxBin, err := getMageboxBinary()
	if err != nil {
		return err
	}

	cli.PrintTitle("Installing MageBox Autostart Service")
	fmt.Println()

	switch p.Type {
	case platform.Linux:
		return installSystemdService(mageboxBin)
	case platform.Darwin:
		return installLaunchdAgent(p, mageboxBin)
	default:
		return fmt.Errorf("unsupported platform: %s", p.Type)
	}
}

func runServiceUninstall(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	cli.PrintTitle("Removing MageBox Autostart Service")
	fmt.Println()

	switch p.Type {
	case platform.Linux:
		return uninstallSystemdService()
	case platform.Darwin:
		return uninstallLaunchdAgent()
	default:
		return fmt.Errorf("unsupported platform: %s", p.Type)
	}
}

func runServiceStatus(cmd *cobra.Command, args []string) error {
	p, err := getPlatform()
	if err != nil {
		return err
	}

	switch p.Type {
	case platform.Linux:
		return statusSystemdService()
	case platform.Darwin:
		return statusLaunchdAgent()
	default:
		return fmt.Errorf("unsupported platform: %s", p.Type)
	}
}

// --- systemd (Linux) ---

func generateSystemdUnit(mageboxBin string) string {
	return fmt.Sprintf(`[Unit]
Description=MageBox Development Environment
After=network.target docker.service
Wants=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=%s service run
ExecStop=%s global stop

[Install]
WantedBy=default.target
`, mageboxBin, mageboxBin)
}

func installSystemdService(mageboxBin string) error {
	dir := systemdUserDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create systemd user directory: %w", err)
	}

	servicePath := systemdServicePath()
	unit := generateSystemdUnit(mageboxBin)

	fmt.Printf("  Writing %s... ", cli.Path(servicePath))
	if err := os.WriteFile(servicePath, []byte(unit), 0644); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to write service file: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Reload systemd
	fmt.Print("  Reloading systemd... ")
	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to reload systemd: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Enable the service
	fmt.Print("  Enabling service... ")
	if err := exec.Command("systemctl", "--user", "enable", systemdServiceName).Run(); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to enable service: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Enable lingering so the user service starts at boot (not just at login)
	fmt.Print("  Enabling user lingering... ")
	if err := exec.Command("loginctl", "enable-linger").Run(); err != nil {
		fmt.Println(cli.Warning("skipped"))
		cli.PrintWarning("Could not enable lingering: %v", err)
		cli.PrintInfo("Service will start on login instead of boot. Enable manually: loginctl enable-linger")
	} else {
		fmt.Println(cli.Success("done"))
	}

	fmt.Println()
	cli.PrintSuccess("MageBox autostart service installed!")
	fmt.Println()
	cli.PrintInfo("The service will start automatically on boot/login.")
	cli.PrintInfo("To start it now: %s", cli.Command("systemctl --user start magebox"))
	cli.PrintInfo("To view logs:    %s", cli.Command("journalctl --user -u magebox"))

	return nil
}

func uninstallSystemdService() error {
	servicePath := systemdServicePath()

	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		cli.PrintInfo("MageBox autostart service is not installed")
		return nil
	}

	// Stop the service
	fmt.Print("  Stopping service... ")
	stopCmd := exec.Command("systemctl", "--user", "stop", systemdServiceName)
	if err := stopCmd.Run(); err != nil {
		fmt.Println(cli.Warning("not running"))
	} else {
		fmt.Println(cli.Success("done"))
	}

	// Disable the service
	fmt.Print("  Disabling service... ")
	if err := exec.Command("systemctl", "--user", "disable", systemdServiceName).Run(); err != nil {
		fmt.Println(cli.Warning("skipped"))
	} else {
		fmt.Println(cli.Success("done"))
	}

	// Remove the service file
	fmt.Print("  Removing service file... ")
	if err := os.Remove(servicePath); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to remove service file: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Reload systemd
	fmt.Print("  Reloading systemd... ")
	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		fmt.Println(cli.Warning("failed"))
	} else {
		fmt.Println(cli.Success("done"))
	}

	fmt.Println()
	cli.PrintSuccess("MageBox autostart service removed")

	return nil
}

func statusSystemdService() error {
	servicePath := systemdServicePath()

	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		fmt.Printf("  %-20s %s\n", "Installed:", cli.Status(false))
		return nil
	}

	fmt.Printf("  %-20s %s\n", "Installed:", cli.Status(true))

	// Check if enabled
	enabledCmd := exec.Command("systemctl", "--user", "is-enabled", systemdServiceName)
	enabledOut, _ := enabledCmd.Output()
	enabled := strings.TrimSpace(string(enabledOut)) == "enabled"
	fmt.Printf("  %-20s %s\n", "Enabled:", cli.Status(enabled))

	// Check if active
	activeCmd := exec.Command("systemctl", "--user", "is-active", systemdServiceName)
	activeOut, _ := activeCmd.Output()
	active := strings.TrimSpace(string(activeOut)) == "active"
	fmt.Printf("  %-20s %s\n", "Active:", cli.Status(active))

	// Check lingering
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("LOGNAME")
	}
	lingerPath := filepath.Join("/var/lib/systemd/linger", user)
	_, lingerErr := os.Stat(lingerPath)
	fmt.Printf("  %-20s %s\n", "Linger:", cli.Status(lingerErr == nil))

	return nil
}

// --- launchd (macOS) ---

func generateLaunchdPlist(p *platform.Platform, mageboxBin string) string {
	logDir := filepath.Join(p.MageBoxDir(), "logs")
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>service</string>
        <string>run</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s/magebox-service.log</string>
    <key>StandardErrorPath</key>
    <string>%s/magebox-service.log</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>
</dict>
</plist>
`, launchdLabel, mageboxBin, logDir, logDir)
}

func installLaunchdAgent(p *platform.Platform, mageboxBin string) error {
	dir := launchdAgentDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	plistPath := launchdPlistPath()
	plist := generateLaunchdPlist(p, mageboxBin)

	// Unload existing if present
	if _, err := os.Stat(plistPath); err == nil {
		_ = exec.Command("launchctl", "unload", plistPath).Run()
	}

	fmt.Printf("  Writing %s... ", cli.Path(plistPath))
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to write plist: %w", err)
	}
	fmt.Println(cli.Success("done"))

	// Load the agent
	fmt.Print("  Loading launch agent... ")
	if err := exec.Command("launchctl", "load", plistPath).Run(); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to load launch agent: %w", err)
	}
	fmt.Println(cli.Success("done"))

	fmt.Println()
	cli.PrintSuccess("MageBox autostart service installed!")
	fmt.Println()
	cli.PrintInfo("The service will start automatically on login.")
	cli.PrintInfo("Logs: %s", cli.Path(filepath.Join(p.MageBoxDir(), "logs", "magebox-service.log")))

	return nil
}

func uninstallLaunchdAgent() error {
	plistPath := launchdPlistPath()

	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		cli.PrintInfo("MageBox autostart service is not installed")
		return nil
	}

	// Unload the agent
	fmt.Print("  Unloading launch agent... ")
	if err := exec.Command("launchctl", "unload", plistPath).Run(); err != nil {
		fmt.Println(cli.Warning("not loaded"))
	} else {
		fmt.Println(cli.Success("done"))
	}

	// Remove the plist
	fmt.Print("  Removing plist... ")
	if err := os.Remove(plistPath); err != nil {
		fmt.Println(cli.Error("failed"))
		return fmt.Errorf("failed to remove plist: %w", err)
	}
	fmt.Println(cli.Success("done"))

	fmt.Println()
	cli.PrintSuccess("MageBox autostart service removed")

	return nil
}

func statusLaunchdAgent() error {
	plistPath := launchdPlistPath()

	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		fmt.Printf("  %-20s %s\n", "Installed:", cli.Status(false))
		return nil
	}

	fmt.Printf("  %-20s %s\n", "Installed:", cli.Status(true))

	// Check if loaded
	listCmd := exec.Command("launchctl", "list", launchdLabel)
	loaded := listCmd.Run() == nil
	fmt.Printf("  %-20s %s\n", "Loaded:", cli.Status(loaded))

	return nil
}

// --- service run (internal command used by systemd/launchd) ---

var serviceRunCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run startup sequence (used by the autostart service)",
	Long:   "Starts global services and all projects. This is called by the systemd/launchd service.",
	Hidden: true,
	RunE:   runServiceRun,
}

func init() {
	serviceCmd.AddCommand(serviceRunCmd)
}

func runServiceRun(cmd *cobra.Command, args []string) error {
	mageboxBin, err := getMageboxBinary()
	if err != nil {
		return err
	}

	fmt.Println("MageBox service starting...")

	// Wait for Docker to be ready (it may still be starting after boot)
	fmt.Print("  Waiting for Docker... ")
	if err := waitForDocker(); err != nil {
		fmt.Println("failed: " + err.Error())
		return fmt.Errorf("docker not available: %w", err)
	}
	fmt.Println("ready")

	// Run magebox global start
	fmt.Println("  Starting global services...")
	globalCmd := exec.Command(mageboxBin, "global", "start")
	globalCmd.Stdout = os.Stdout
	globalCmd.Stderr = os.Stderr
	if err := globalCmd.Run(); err != nil {
		fmt.Printf("  Global start failed: %v\n", err)
		// Continue anyway — project start may still partially work
	}

	// Run magebox start --all
	fmt.Println("  Starting all projects...")
	startCmd := exec.Command(mageboxBin, "start", "--all")
	startCmd.Stdout = os.Stdout
	startCmd.Stderr = os.Stderr
	if err := startCmd.Run(); err != nil {
		fmt.Printf("  Project start failed: %v\n", err)
	}

	fmt.Println("MageBox service startup complete")
	return nil
}

// waitForDocker waits up to 60 seconds for the Docker daemon to become available
func waitForDocker() error {
	maxAttempts := 30
	if runtime.GOOS == "darwin" {
		// On macOS, Docker Desktop may take a while to start
		maxAttempts = 60
	}

	for i := 0; i < maxAttempts; i++ {
		if exec.Command("docker", "info").Run() == nil {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("docker did not become ready within timeout")
}

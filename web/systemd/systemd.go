// Package systemd renders, installs and removes systemd unit files for the couchness web UI.
package systemd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// systemDir is where system unit files are written; overridable in tests.
var systemDir = "/etc/systemd/system"

// execCommand is the command factory; overridable in tests.
var execCommand = exec.Command

// Options describes the unit to render/install.
type Options struct {
	Name        string   // unit name without ".service", e.g. "couchness-web"
	Executable  string   // absolute path to the couchness binary
	ConfigDir   string   // passed as --config-dir
	Address     string   // passed as --addr
	Auth        string   // passed as --auth when non-empty
	User        bool     // true -> user unit (~/.config/systemd/user), false -> system unit
	RunAs       string   // User= for system units (ignored when User is true)
	EnvFile     string   // EnvironmentFile=-<path> when non-empty
	Environment []string // Environment=KEY=VALUE lines
}

// Unit renders the systemd unit file content.
func Unit(options Options) string {
	var b strings.Builder
	b.WriteString("[Unit]\n")
	b.WriteString("Description=Couchness web UI\n")
	b.WriteString("After=network-online.target\n")
	b.WriteString("Wants=network-online.target\n")
	b.WriteString("\n[Service]\n")
	b.WriteString("Type=simple\n")
	b.WriteString("ExecStart=" + execStart(options) + "\n")
	b.WriteString("Restart=on-failure\n")
	b.WriteString("RestartSec=5\n")
	if !options.User && options.RunAs != "" {
		b.WriteString("User=" + options.RunAs + "\n")
	}
	if options.EnvFile != "" {
		b.WriteString("EnvironmentFile=-" + options.EnvFile + "\n")
	}
	for _, entry := range options.Environment {
		b.WriteString("Environment=" + quote(entry) + "\n")
	}
	b.WriteString("\n[Install]\n")
	wantedBy := "multi-user.target"
	if options.User {
		wantedBy = "default.target"
	}
	b.WriteString("WantedBy=" + wantedBy + "\n")
	return b.String()
}

// UnitPath returns the unit file path: /etc/systemd/system/<name>.service for
// system units or $XDG_CONFIG_HOME|~/.config /systemd/user/<name>.service for user units.
func UnitPath(name string, user bool) (string, error) {
	if !user {
		return filepath.Join(systemDir, name+".service"), nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot locate the user configuration directory: %w", err)
	}
	return filepath.Join(configDir, "systemd", "user", name+".service"), nil
}

// Install writes the unit, runs daemon-reload and enable (--now when start).
// On permission errors it returns: "cannot write <path>: <err> (run with sudo for a system service, or pass --user)".
func Install(options Options, start bool) error {
	if options.Name == "" {
		options.Name = "couchness-web"
	}
	if options.Address == "" {
		options.Address = ":8085"
	}
	path, err := UnitPath(options.Name, options.User)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return permissionHint(err, path)
	}
	mode := fs.FileMode(0644)
	if options.Auth != "" {
		mode = 0600
	}
	if err := os.WriteFile(path, []byte(Unit(options)), mode); err != nil {
		return permissionHint(err, path)
	}
	if err := systemctl(options.User, "daemon-reload"); err != nil {
		return permissionHint(err, path)
	}
	if start {
		if err := systemctl(options.User, "enable", "--now", options.Name); err != nil {
			return permissionHint(err, path)
		}
	} else {
		if err := systemctl(options.User, "enable", options.Name); err != nil {
			return permissionHint(err, path)
		}
	}
	return nil
}

// Uninstall runs `disable --now` (errors ignored), removes the unit file (missing file ok), daemon-reload.
func Uninstall(name string, user bool) error {
	_ = systemctl(user, "disable", "--now", name)
	path, err := UnitPath(name, user)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return systemctl(user, "daemon-reload")
}

// Status runs `systemctl [--user] status <name>` with stdout/stderr passed through; returns the exec error.
func Status(name string, user bool) error {
	return systemctl(user, "status", name)
}

func systemctl(user bool, args ...string) error {
	full := make([]string, 0, len(args)+1)
	if user {
		full = append(full, "--user")
	}
	full = append(full, args...)
	cmd := execCommand("systemctl", full...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// quote wraps s in double quotes when it contains whitespace or '"', escaping '"' as '\"'.
func quote(s string) string {
	if !strings.ContainsAny(s, " \t\n\"") {
		return s
	}
	escaped := strings.ReplaceAll(s, `"`, `\"`)
	return `"` + escaped + `"`
}

// execStart renders the ExecStart command line.
func execStart(o Options) string {
	start := quote(o.Executable) +
		" --config-dir " + quote(o.ConfigDir) +
		" web run --addr " + quote(o.Address)
	if o.Auth != "" {
		start += " --auth " + quote(o.Auth)
	}
	return start
}

func permissionHint(err error, path string) error {
	if errors.Is(err, fs.ErrPermission) {
		return fmt.Errorf("cannot write %s: %w (run with sudo or use --user)", path, err)
	}
	return err
}

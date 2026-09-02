package systemd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnitSystem(t *testing.T) {
	options := Options{
		Name:        "couchness-web",
		Executable:  "/usr/local/bin/couchness",
		ConfigDir:   "/home/bob/.couchness",
		Address:     ":8085",
		Auth:        "bob:s3cret",
		RunAs:       "bob",
		EnvFile:     "/etc/couchness.env",
		Environment: []string{"COUCHNESS_OMDB_API_KEY=abc"},
	}
	got := Unit(options)
	if !strings.Contains(got, "ExecStart=/usr/local/bin/couchness --config-dir /home/bob/.couchness web run --addr :8085 --auth bob:s3cret") {
		t.Errorf("ExecStart wrong:\n%s", got)
	}
	if !strings.Contains(got, "User=bob\n") {
		t.Errorf("missing User=bob line:\n%s", got)
	}
	if !strings.Contains(got, "Restart=on-failure") {
		t.Errorf("missing Restart=on-failure:\n%s", got)
	}
	if !strings.Contains(got, "EnvironmentFile=-/etc/couchness.env") {
		t.Errorf("missing EnvironmentFile:\n%s", got)
	}
	if !strings.Contains(got, "Environment=COUCHNESS_OMDB_API_KEY=abc") {
		t.Errorf("missing Environment line:\n%s", got)
	}
	if !strings.Contains(got, "WantedBy=multi-user.target") {
		t.Errorf("missing WantedBy=multi-user.target:\n%s", got)
	}
}

func TestUnitUser(t *testing.T) {
	options := Options{
		Name:       "couchness-web",
		Executable: "/usr/local/bin/couchness",
		ConfigDir:  "/home/bob/.couchness",
		Address:    ":8085",
		User:       true,
	}
	got := Unit(options)
	if strings.Contains(got, "User=") {
		t.Errorf("user unit must not have a User= line:\n%s", got)
	}
	if !strings.Contains(got, "WantedBy=default.target") {
		t.Errorf("missing WantedBy=default.target:\n%s", got)
	}
	if strings.Contains(got, "--auth") {
		t.Errorf("no --auth expected when Auth empty:\n%s", got)
	}
}

func TestUnitQuotesSpaces(t *testing.T) {
	options := Options{
		Name:       "couchness-web",
		Executable: "/usr/local/bin/couchness",
		ConfigDir:  "/home/bob/my dir",
		Address:    ":8085",
	}
	got := Unit(options)
	if !strings.Contains(got, `"--config-dir "/home/bob/my dir"`) && !strings.Contains(got, `--config-dir "/home/bob/my dir"`) {
		t.Errorf("ConfigDir with spaces must be quoted:\n%s", got)
	}
	if !strings.Contains(got, `"/home/bob/my dir"`) {
		t.Errorf("missing quoted path:\n%s", got)
	}
}

func TestUnitPath(t *testing.T) {
	t.Setenv("HOME", "/home/bob")
	t.Setenv("XDG_CONFIG_HOME", "")

	system, err := UnitPath("x", false)
	if err != nil {
		t.Fatalf("UnitPath system: %v", err)
	}
	if system != "/etc/systemd/system/x.service" {
		t.Errorf("system unit path = %q, want /etc/systemd/system/x.service", system)
	}

	user, err := UnitPath("x", true)
	if err != nil {
		t.Fatalf("UnitPath user: %v", err)
	}
	if !strings.HasSuffix(user, "/.config/systemd/user/x.service") {
		t.Errorf("user unit path = %q, want suffix /.config/systemd/user/x.service", user)
	}
}

func TestInstallWritesFileAndCallsSystemctl(t *testing.T) {
	dir := t.TempDir()
	originalSystemDir := systemDir
	systemDir = dir
	defer func() { systemDir = originalSystemDir }()

	type recorded struct {
		name string
		args []string
	}
	var calls []recorded
	originalExecCommand := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		calls = append(calls, recorded{name: name, args: append([]string{}, args...)})
		return exec.Command("true")
	}
	defer func() { execCommand = originalExecCommand }()

	options := Options{
		Name:       "couchness-web",
		Executable: "/usr/local/bin/couchness",
		ConfigDir:  "/home/bob/.couchness",
		Address:    ":8085",
		Auth:       "bob:s3cret",
	}
	if err := Install(options, true); err != nil {
		t.Fatalf("Install: %v", err)
	}

	unitPath := filepath.Join(dir, "couchness-web.service")
	data, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("unit file not written: %v", err)
	}
	if len(data) == 0 {
		t.Error("unit file is empty")
	}
	info, err := os.Stat(unitPath)
	if err != nil {
		t.Fatalf("stat unit: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("unit mode = %o, want 600 (Auth set)", got)
	}

	if len(calls) != 2 {
		t.Fatalf("expected 2 systemctl calls, got %d: %+v", len(calls), calls)
	}
	if calls[0].name != "systemctl" || !equalStrings(calls[0].args, []string{"daemon-reload"}) {
		t.Errorf("first call = %s %v, want systemctl daemon-reload", calls[0].name, calls[0].args)
	}
	if calls[1].name != "systemctl" || !equalStrings(calls[1].args, []string{"enable", "--now", "couchness-web"}) {
		t.Errorf("second call = %s %v, want systemctl enable --now couchness-web", calls[1].name, calls[1].args)
	}
}

func TestInstallNoStartSkipsNow(t *testing.T) {
	dir := t.TempDir()
	originalSystemDir := systemDir
	systemDir = dir
	defer func() { systemDir = originalSystemDir }()

	var args [][]string
	originalExecCommand := execCommand
	execCommand = func(name string, a ...string) *exec.Cmd {
		args = append(args, append([]string{}, a...))
		return exec.Command("true")
	}
	defer func() { execCommand = originalExecCommand }()

	options := Options{Name: "check", Executable: "/bin/true", ConfigDir: "/tmp/x", Address: ":1"}
	if err := Install(options, false); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(args) != 2 || !equalStrings(args[1], []string{"enable", "check"}) {
		t.Errorf("expected enable without --now, got %+v", args)
	}
	info, err := os.Stat(filepath.Join(dir, "check.service"))
	if err != nil {
		t.Fatalf("stat unit: %v", err)
	}
	if got := info.Mode().Perm(); got != 0644 {
		t.Errorf("unit mode = %o, want 644 (no Auth)", got)
	}
}

func TestQuote(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"plain", "plain"},
		{"my dir", `"my dir"`},
		{`he"llo`, `"he\"llo"`},
	}
	for _, c := range cases {
		if got := quote(c.in); got != c.want {
			t.Errorf("quote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

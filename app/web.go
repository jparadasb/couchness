package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"syscall"

	"github.com/highercomve/couchness/storage"
	"github.com/highercomve/couchness/web"
	"github.com/highercomve/couchness/web/systemd"
	cli "github.com/urfave/cli/v2"
)

// Web manages the web UI: run it, or install/uninstall/inspect it as a systemd service.
func Web() *cli.Command {
	return &cli.Command{
		Name:  "web",
		Usage: "run or install the web UI",
		Subcommands: []*cli.Command{
			webRun(),
			webInstall(),
			webUninstall(),
			webStatus(),
		},
	}
}

func webRun() *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: "run the web UI server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "addr",
				Usage:   "listen address",
				Value:   web.DefaultAddress,
				EnvVars: []string{"COUCHNESS_WEB_ADDR"},
			},
			&cli.StringFlag{
				Name:    "auth",
				Usage:   "user:password for HTTP basic auth; empty disables it",
				Value:   "",
				EnvVars: []string{"COUCHNESS_WEB_AUTH"},
			},
		},
		Action: func(c *cli.Context) error {
			ctx, cancel := context.WithCancel(c.Context)
			defer cancel()
			signals := make(chan os.Signal, 1)
			signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
			defer signal.Stop(signals)
			go func() {
				select {
				case <-signals:
					cancel()
				case <-ctx.Done():
				}
			}()

			addr := c.String("addr")
			fmt.Printf("Couchness web UI listening on %s\n", addr)
			if err := web.Run(ctx, addr, web.Options{
				Auth:    c.String("auth"),
				Version: c.App.Version,
			}); err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}

func webInstall() *cli.Command {
	return &cli.Command{
		Name:  "install",
		Usage: "install the web UI as a systemd service",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "addr",
				Usage:   "listen address written into the unit",
				Value:   web.DefaultAddress,
				EnvVars: []string{"COUCHNESS_WEB_ADDR"},
			},
			&cli.StringFlag{
				Name:    "auth",
				Usage:   "user:password for HTTP basic auth written into the unit",
				Value:   "",
				EnvVars: []string{"COUCHNESS_WEB_AUTH"},
			},
			&cli.BoolFlag{
				Name:  "user",
				Usage: "install a user unit (~/.config/systemd/user) instead of a system unit",
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "unit name",
				Value: "couchness-web",
			},
			&cli.StringFlag{
				Name:  "run-as",
				Usage: "User= for system units",
				Value: defaultRunAs(),
			},
			&cli.StringFlag{
				Name:  "env-file",
				Usage: "EnvironmentFile=-<path> written into the unit",
				Value: "",
			},
			&cli.StringSliceFlag{
				Name:  "env",
				Usage: "Environment=KEY=VALUE line, repeatable",
			},
			&cli.BoolFlag{
				Name:  "no-start",
				Usage: "enable the unit only, do not start it",
			},
			&cli.BoolFlag{
				Name:  "print",
				Usage: "print the unit to stdout and exit without touching the system",
			},
		},
		Action: func(c *cli.Context) error {
			exe, err := os.Executable()
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			if exe, err = filepath.EvalSymlinks(exe); err != nil {
				return cli.Exit(err.Error(), 1)
			}

			configDir, err := resolveConfigDir(c)
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

			options := systemd.Options{
				Name:        c.String("name"),
				Executable:  exe,
				ConfigDir:   configDir,
				Address:     c.String("addr"),
				Auth:        c.String("auth"),
				User:        c.Bool("user"),
				RunAs:       c.String("run-as"),
				EnvFile:     c.String("env-file"),
				Environment: c.StringSlice("env"),
			}

			if c.Bool("print") {
				fmt.Print(systemd.Unit(options))
				return nil
			}

			if err := systemd.Install(options, !c.Bool("no-start")); err != nil {
				return cli.Exit(err.Error(), 1)
			}

			path, _ := systemd.UnitPath(options.Name, options.User)
			fmt.Printf("Installed %s\n", path)
			fmt.Printf("Check it with: couchness web status%s\n", userFlagHint(options.User))
			if options.User {
				fmt.Println("Hint: run `loginctl enable-linger` so the service starts at boot without login.")
			}
			return nil
		},
	}
}

func webUninstall() *cli.Command {
	return &cli.Command{
		Name:  "uninstall",
		Usage: "remove the web UI systemd service",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "user",
				Usage: "remove a user unit instead of a system unit",
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "unit name",
				Value: "couchness-web",
			},
		},
		Action: func(c *cli.Context) error {
			name := c.String("name")
			userMode := c.Bool("user")
			if err := systemd.Uninstall(name, userMode); err != nil {
				return cli.Exit(err.Error(), 1)
			}
			fmt.Printf("Removed %s\n", name)
			return nil
		},
	}
}

func webStatus() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "show the web UI systemd service status",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "user",
				Usage: "inspect a user unit instead of a system unit",
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "unit name",
				Value: "couchness-web",
			},
		},
		Action: func(c *cli.Context) error {
			if err := systemd.Status(c.String("name"), c.Bool("user")); err != nil {
				return cli.Exit(err.Error(), 1)
			}
			return nil
		},
	}
}

// userFlagHint returns " --user" when user mode is on.
func userFlagHint(user bool) string {
	if user {
		return " --user"
	}
	return ""
}

// resolveConfigDir picks the --config-dir value written into the unit:
//  1. if the global --config-dir flag was set -> storage.Db.Directory
//  2. else if SUDO_USER is set -> that user's home + "/.couchness"
//  3. else -> storage.Db.Directory
func resolveConfigDir(c *cli.Context) (string, error) {
	if !c.IsSet("config-dir") && os.Getenv("SUDO_USER") != "" {
		sudoUser, err := user.Lookup(os.Getenv("SUDO_USER"))
		if err != nil {
			return "", err
		}
		return sudoUser.HomeDir + "/.couchness", nil
	}
	return storage.Db.Directory, nil
}

// defaultRunAs returns $SUDO_USER, else user.Current().Username, else "".
func defaultRunAs() string {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		return sudoUser
	}
	if current, err := user.Current(); err == nil {
		return current.Username
	}
	return ""
}

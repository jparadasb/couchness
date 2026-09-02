package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/storage"
	telegrambot "github.com/highercomve/couchness/telegram"
	"github.com/urfave/cli/v2"
)

// Telegram manages and runs Telegram integration.
func Telegram() *cli.Command {
	return &cli.Command{
		Name:  "telegram",
		Usage: "manage Telegram integration",
		Subcommands: []*cli.Command{
			telegramSetup(),
			telegramDisable(),
			telegramRun(),
			telegramStatus(),
			telegramTest(),
			telegramUsers(),
		},
	}
}

func telegramDisable() *cli.Command {
	return &cli.Command{
		Name:  "disable",
		Usage: "disable Telegram integration",
		Action: func(c *cli.Context) error {
			configuration, err := storage.GetTelegramConfiguration()
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			configuration.Enabled = false
			if err := storage.SaveTelegramConfiguration(configuration); err != nil {
				return cli.Exit(err.Error(), 1)
			}
			fmt.Println("Telegram integration disabled.")
			return nil
		},
	}
}

func telegramSetup() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "validate bot token and enable Telegram",
		Flags: []cli.Flag{
			&cli.Int64Flag{Name: "owner-id", Usage: "initial owner's numeric Telegram user ID"},
		},
		Action: func(c *cli.Context) error {
			bot, err := telegrambot.New(c.Context, telegrambot.TokenFromEnvironment())
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			configuration, err := storage.GetTelegramConfiguration()
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			configuration.Enabled = true
			ownerID := c.Int64("owner-id")
			if ownerID != 0 {
				configuration.Users[strconv.FormatInt(ownerID, 10)] = &models.TelegramUser{
					ID:      ownerID,
					Role:    models.TelegramRoleOwner,
					AddedAt: time.Now().UTC(),
				}
			}
			if err := storage.SaveTelegramConfiguration(configuration); err != nil {
				return cli.Exit(err.Error(), 1)
			}
			fmt.Printf("Telegram enabled for @%s.\n", bot.Username())
			if ownerID == 0 && len(configuration.Users) == 0 {
				fmt.Println("No owner configured. Start bot, send /start, then add reported ID with telegram users add --role owner.")
			}
			return nil
		},
	}
}

func telegramRun() *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: "run Telegram bot using long polling",
		Action: func(c *cli.Context) error {
			configuration, err := storage.GetTelegramConfiguration()
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			if !configuration.Enabled {
				return cli.Exit("Telegram integration is disabled; run telegram setup", 1)
			}
			bot, err := telegrambot.New(c.Context, telegrambot.TokenFromEnvironment())
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}

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

			fmt.Printf("Telegram bot @%s running.\n", bot.Username())
			return bot.Run(ctx)
		},
	}
}

func telegramStatus() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "show Telegram configuration status",
		Action: func(c *cli.Context) error {
			configuration, err := storage.GetTelegramConfiguration()
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			fmt.Printf("Enabled: %t\nAuthorized users: %d\nActive invites: %d\nToken present: %t\n",
				configuration.Enabled,
				len(configuration.Users),
				len(configuration.Invites),
				telegrambot.TokenFromEnvironment() != "",
			)
			return nil
		},
	}
}

func telegramTest() *cli.Command {
	return &cli.Command{
		Name:  "test",
		Usage: "validate Telegram bot token",
		Action: func(c *cli.Context) error {
			bot, err := telegrambot.New(c.Context, telegrambot.TokenFromEnvironment())
			if err != nil {
				return cli.Exit(err.Error(), 1)
			}
			fmt.Printf("Connected to @%s.\n", bot.Username())
			return nil
		},
	}
}

func telegramUsers() *cli.Command {
	return &cli.Command{
		Name:  "users",
		Usage: "manage authorized Telegram users",
		Subcommands: []*cli.Command{
			{
				Name:  "list",
				Usage: "list authorized Telegram users",
				Action: func(c *cli.Context) error {
					configuration, err := storage.GetTelegramConfiguration()
					if err != nil {
						return cli.Exit(err.Error(), 1)
					}
					ids := make([]string, 0, len(configuration.Users))
					for id := range configuration.Users {
						ids = append(ids, id)
					}
					sort.Strings(ids)
					for _, id := range ids {
						fmt.Printf("%s\t%s\n", id, configuration.Users[id].Role)
					}
					return nil
				},
			},
			{
				Name:      "add",
				Usage:     "authorize a numeric Telegram user ID",
				ArgsUsage: "<telegram_user_id>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "role", Value: models.TelegramRoleUser, Usage: "owner, admin, user, or viewer"},
				},
				Action: func(c *cli.Context) error {
					arguments := c.Args().Slice()
					id, err := strconv.ParseInt(c.Args().First(), 10, 64)
					if err != nil || id <= 0 {
						return cli.Exit("Telegram user ID must be a positive number", 1)
					}
					role := c.String("role")
					if len(arguments) >= 3 && arguments[1] == "--role" {
						role = arguments[2]
					} else if len(arguments) >= 2 && strings.HasPrefix(arguments[1], "--role=") {
						role = strings.TrimPrefix(arguments[1], "--role=")
					}
					if !models.ValidTelegramRole(role) {
						return cli.Exit("role must be owner, admin, user, or viewer", 1)
					}
					if err := storage.AddTelegramUser(id, role); err != nil {
						return cli.Exit(err.Error(), 1)
					}
					fmt.Printf("Authorized Telegram user %d as %s.\n", id, role)
					return nil
				},
			},
			{
				Name:      "remove",
				Usage:     "revoke a Telegram user",
				ArgsUsage: "<telegram_user_id>",
				Action: func(c *cli.Context) error {
					id, err := strconv.ParseInt(c.Args().First(), 10, 64)
					if err != nil || id <= 0 {
						return cli.Exit("Telegram user ID must be a positive number", 1)
					}
					if err := storage.RemoveTelegramUser(id); err != nil {
						return cli.Exit(err.Error(), 1)
					}
					fmt.Printf("Revoked Telegram user %d.\n", id)
					return nil
				},
			},
		},
	}
}

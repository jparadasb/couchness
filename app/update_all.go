package app

import (
	"fmt"

	"github.com/highercomve/couchness/common"
	"github.com/urfave/cli/v2"
)

// UpdateAll scan your show media and start shows download
func UpdateAll() *cli.Command {
	return &cli.Command{
		Name:        "update-all",
		Aliases:     []string{"ua"},
		ArgsUsage:   "",
		Usage:       "update all your shows",
		HelpName:    "",
		Description: "update all your shows",
		Action: func(c *cli.Context) error {
			fmt.Println("Updating database...")
			err := common.UpdateAll(func(message string) {
				fmt.Println(message)
			})
			if err != nil {
				return cli.Exit(err.Error(), 0)
			}

			fmt.Printf("\n\r\n\rAll Show now are updated! \n\r")
			return nil
		},
	}
}

package cmd

import (
	"dp-cli/config"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func spew() *cobra.Command {
	command := &cobra.Command{
		Use:   "spew",
		Short: "log out some useful debugging info",
	}

	command.AddCommand(logConfig())
	return command
}

func logConfig() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "spew out your dp-cli config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			b, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return err
			}

			fmt.Println(string(b))

			ip, err := config.GetMyIP()
			if err != nil {
				return err
			}

			fmt.Println(ip)
			return nil
		},
	}
}

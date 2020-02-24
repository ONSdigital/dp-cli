package cmd

import (
	"dp-cli/config"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func debugConfig() *cobra.Command {
	return &cobra.Command{
		Use:   "debug",
		Short: "print out cli configuration as json",
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

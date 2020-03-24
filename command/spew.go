package command

import (
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"

	"github.com/spf13/cobra"
)

func spew() *cobra.Command {
	command := &cobra.Command{
		Use:   "spew",
		Short: "Log out some useful debugging info",
	}

	command.AddCommand(logConfig())
	return command
}

func logConfig() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "spew out your dp-cli config",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := config.Dump()
			if err != nil {
				return err
			}

			out.InfoFHighlight("configuration:\n%s", string(data))
			return nil
		},
	}
}

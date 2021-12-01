package command

import (
	"github.com/ONSdigital/dp-cli/out"

	"github.com/spf13/cobra"
)

func versionSubCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the app version",
		Run: func(cmd *cobra.Command, args []string) {
			out.Info(AppVersion)
		},
	}
}

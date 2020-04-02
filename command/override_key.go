package command

import (
	"time"

	"github.com/ONSdigital/dp-cli/out"
	"github.com/spf13/cobra"
)

func overrideKey() *cobra.Command {
	return &cobra.Command{
		Use:   "override-key",
		Short: "Generates an overrideKey to bypass the Florence dataset version validation step when approving a collection",
		RunE: func(cmd *cobra.Command, args []string) error {
			now := time.Now()
			midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.Local)
			out.Highlight(out.INFO, "%v", int(midnight.Sub(now).Minutes()))
			return nil
		},
	}
}

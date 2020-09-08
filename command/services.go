package command

import (
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/services"

	"github.com/spf13/cobra"
)

func servicesCommand(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "services",
		Short: "Manage services",
	}
	opts := config.WithOpts{
		Interactive:     cmd.PersistentFlags().BoolP("interactive", "i", false, "interactive prompting"),
		LimitWeb:        cmd.PersistentFlags().BoolP("web", "w", false, "limit to web services"),
		LimitPublishing: cmd.PersistentFlags().BoolP("pub", "p", false, "limit to publishing services"),
		Verbose:         cmd.PersistentFlags().CountP("verbose", "v", "increase verbosity"),
	}

	subCommands := []*cobra.Command{
		listCommand(cfg, opts),
		cloneCommand(cfg, opts),
		pullCommand(cfg, opts),
	}

	cmd.AddCommand(subCommands...)
	return cmd
}

// build the sub command - has sub commands for each action
func listCommand(cfg *config.Config, opts config.WithOpts) *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "list services",
		RunE: func(cmd *cobra.Command, args []string) error {
			return services.List(cfg, opts, args)
		},
	}
	return c
}

// build the sub command - has sub commands for each action
func cloneCommand(cfg *config.Config, opts config.WithOpts) *cobra.Command {
	c := &cobra.Command{
		Use:   "clone",
		Short: "clone services",
		RunE: func(cmd *cobra.Command, args []string) error {
			return services.Clone(cfg, opts, args)
		},
	}
	return c
}

// build the sub command - has sub commands for each action
func pullCommand(cfg *config.Config, opts config.WithOpts) *cobra.Command {
	c := &cobra.Command{
		Use:   "pull",
		Short: "git pull service repos",
		RunE: func(cmd *cobra.Command, args []string) error {
			return services.Pull(cfg, opts, args)
		},
	}
	return c
}

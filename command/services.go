package command

import (
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/services"

	"github.com/spf13/cobra"
)

func servicesCommand(cfg *config.Config, appVersion string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "services",
		Short: "Manage services",
	}
	opts := config.WithOpts{
		Interactive:     cmd.PersistentFlags().BoolP("interactive", "i", false, "interactive prompting"),
		Itermaton:       cmd.PersistentFlags().Bool("itermaton", false, "use iTermaton output"),
		LimitWeb:        cmd.PersistentFlags().BoolP("web", "w", false, "limit to web services"),
		LimitPublishing: cmd.PersistentFlags().BoolP("pub", "p", false, "limit to publishing services"),
		Verbose:         cmd.PersistentFlags().CountP("verbose", "v", "increase verbosity"),
		Skip:            cmd.PersistentFlags().StringArrayP("skip", "s", nil, "skip service by name"),
		Tags:            cmd.PersistentFlags().StringArrayP("tags", "t", nil, "limit services by tag"),
		AppVersion:      appVersion,
	}

	subCommands := []*cobra.Command{
		listCommand(cfg, opts),
		startCommand(cfg, opts),
		cloneCommand(cfg, opts),
		execCommand(cfg, opts),
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
func startCommand(cfg *config.Config, opts config.WithOpts) *cobra.Command {
	c := &cobra.Command{
		Use:   "start",
		Short: "start given service(s)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return services.Start(cfg, opts, args)
		},
	}
	return c
}

// build the sub command - has sub commands for each action
func cloneCommand(cfg *config.Config, opts config.WithOpts) *cobra.Command {
	c := &cobra.Command{
		Use:   "clone",
		Short: "clone the repositories for the given service(s)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return services.Clone(cfg, opts, args)
		},
	}
	return c
}

// build the sub command - has sub commands for each action
func execCommand(cfg *config.Config, opts config.WithOpts) *cobra.Command {
	c := &cobra.Command{
		Use:   "exec",
		Short: "run a given command in all selected service directories",
		Long: "the arguments to this command are the command to run in each repo " +
			"directory (remember to follow `exec` with `--` to allow any flags to be " +
			"passed to that command), e.g. `dp services exec -- git pull --prune`",
		RunE: func(cmd *cobra.Command, args []string) error {
			return services.Exec(cfg, opts, args)
		},
	}
	return c
}

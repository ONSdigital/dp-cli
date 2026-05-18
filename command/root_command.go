package command

import (
	"context"

	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"
	"github.com/spf13/cobra"
)

var (
	root       *cobra.Command
	appVersion = "development"
)

// Load will load the sub-commands
func Load(ctx context.Context, cfg *config.Config) (*cobra.Command, error) {

	root = &cobra.Command{
		Use:   "dp",
		Short: "dp is a command-line client providing handy helper tools for ONS Dissemination Platform software engineers",
		// TODO: The following arg as it makes the output cleaner on errors, but needs regression testing
		// SilenceUsage: true, //silence usage when an error occurs
	}

	// register the root sub-commands.
	subCommands, err := getSubCommands(ctx, cfg)
	if err != nil {
		return nil, err
	}

	root.AddCommand(subCommands...)
	return root, nil
}

func getSubCommands(ctx context.Context, cfg *config.Config) ([]*cobra.Command, error) {
	subCommands := []*cobra.Command{
		versionSubCommand(),
		cleanSubCommand(cfg),
		importDataSubCommand(ctx, cfg),
		createRepoSubCommand(),
		generateProjectSubCommand(),
		spew(),
		remoteAccess(ctx, cfg),
		eksCommand(ctx, cfg),
		overrideKey(),
	}

	ssh, err := sshCommand(ctx, cfg)
	if err != nil {
		out.WarnFHighlight("warning: failed to initialise ssh sub-commands: %s", err)
	} else {
		subCommands = append(subCommands, ssh)
	}

	scp, err := scpCommand(ctx, cfg)
	if err != nil {
		out.WarnFHighlight("warning: failed to initialise scp sub-commands: %s", err)
	} else {
		subCommands = append(subCommands, scp)
	}

	return subCommands, nil
}

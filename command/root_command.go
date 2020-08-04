package command

import (
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"
	"github.com/spf13/cobra"
)

var (
	root *cobra.Command

	onsDigitalPath       string
	hierarchyBuilderPath string
	codeListScriptsPath  string
	appVersion           = "development"
)

// Load will load the sub-commands
func Load(cfg *config.Config) (*cobra.Command, error) {

	root = &cobra.Command{
		Use:   "dp",
		Short: "dp is a command-line client providing handy helper tools for ONS Digital Publishing software engineers",
	}

	// register the root sub-commands.
	subCommands, err := getSubCommands(cfg)
	if err != nil {
		return nil, err
	}

	root.AddCommand(subCommands...)
	return root, nil
}

func getSubCommands(cfg *config.Config) ([]*cobra.Command, error) {
	subCommands := []*cobra.Command{
		versionSubCommand(),
		cleanSubCommand(cfg),
		importDataSubCommand(cfg),
		createRepoSubCommand(),
		generateProjectSubCommand(),
		spew(),
		remoteAccess(cfg),
		overrideKey(),
	}

	ssh, err := sshCommand(cfg)
	if err != nil {
		out.WarnFHighlight("warning: failed to initialise ssh sub-commands: %s", err)
	} else {
		subCommands = append(subCommands, ssh)
	}

	scp, err := scpCommand(cfg)
	if err != nil {
		out.WarnFHighlight("warning: failed to initialise scp sub-commands: %s", err)
	} else {
		subCommands = append(subCommands, scp)
	}

	return subCommands, nil
}

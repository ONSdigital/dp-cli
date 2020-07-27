package command

import (
	"math/rand"
	"time"

	"github.com/ONSdigital/dp-cli/config"
	"github.com/spf13/cobra"
)

var (
	root *cobra.Command

	r                    *rand.Rand
	onsDigitalPath       string
	hierarchyBuilderPath string
	codeListScriptsPath  string
	appVersion           = "development"
)

// Load will load the sub-commands
func Load(cfg *config.Config) (*cobra.Command, error) {
	s1 := rand.NewSource(time.Now().UnixNano())
	r = rand.New(s1)

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
		return nil, err
	}

	subCommands = append(subCommands, ssh)
	return subCommands, nil
}

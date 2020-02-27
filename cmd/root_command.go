package cmd

import (
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/ONSdigital/dp-cli/config"
	"github.com/spf13/cobra"
)

var (
	root *cobra.Command

	r                    *rand.Rand
	goPath               string
	onsDigitalPath       string
	hierarchyBuilderPath string
	codeListScriptsPath  string
	appVersion           = "development"
)

func Load(cfg *config.Config) (*cobra.Command, error) {
	s1 := rand.NewSource(time.Now().UnixNano())
	r = rand.New(s1)

	goPath = os.Getenv("GOPATH")

	onsDigitalPath = filepath.Join(goPath, "src/github.com/ONSdigital")

	hierarchyBuilderPath = filepath.Join(onsDigitalPath, "dp-hierarchy-builder/cypher-scripts")

	codeListScriptsPath = filepath.Join(onsDigitalPath, "dp-code-list-scripts/code-list-scripts")

	root = &cobra.Command{
		Use:   "dp-cli",
		Short: "dp-cli provides util functions for developers in ONS Digital Publishing",
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
	}

	ssh, err := sshCommand(cfg)
	if err != nil {
		return nil, err
	}

	subCommands = append(subCommands, ssh)
	return subCommands, nil
}

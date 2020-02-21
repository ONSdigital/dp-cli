package cmd

import (
	"github.com/ONSdigital/dp-cli/config"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	root *cobra.Command

	r                    *rand.Rand
	goPath               string
	onsDigitalPath       string
	hierarchyBuilderPath string
	codeListScriptsPath  string
	appVersion           string
)

func Load(cfg *config.Config) *cobra.Command {
	s1 := rand.NewSource(time.Now().UnixNano())
	r = rand.New(s1)

	appVersion = "v0.0.1"

	goPath = os.Getenv("GOPATH")

	onsDigitalPath = filepath.Join(goPath, "src/github.com/ONSdigital")

	hierarchyBuilderPath = filepath.Join(onsDigitalPath, "dp-hierarchy-builder/cypher-scripts")

	codeListScriptsPath = filepath.Join(onsDigitalPath, "dp-code-list-scripts/code-list-scripts")

	root = &cobra.Command{
		Use:   "dp-cli",
		Short: "dp-cli provides util functions for developers in ONS Digital Publishing",
	}

	// register the root sub-commands.
	root.AddCommand(
		versionSubCommand(),
		cleanSubCommand(cfg),
		importDataSubCommand(cfg),
		createRepoSubCommand(),
		generateProjectSubCommand(),
	)

	return root
}

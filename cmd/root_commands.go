package cmd

import (
	appgen "dp-utils/app-generation"
	"dp-utils/config"
	"dp-utils/customisemydata"
	"dp-utils/out"
	"dp-utils/repocreation"
	"dp-utils/zebedee"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	Root       *cobra.Command
	Version    *cobra.Command
	Clean      *cobra.Command
	Import     *cobra.Command
	Generate   *cobra.Command
	CreateRepo *cobra.Command

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

	Version = &cobra.Command{
		Use:   "version",
		Short: "Print the app version",
		Run: func(cmd *cobra.Command, args []string) {
			out.Info(appVersion)
		},
	}

	Clean = &cobra.Command{
		Use:   "clean",
		Short: "Clean/Delete data from your local environment",
	}
	Clean.AddCommand(tearDownCustomiseMyData(cfg), clearCollections())

	Import = &cobra.Command{
		Use:   "import",
		Short: "ImportData your local developer environment",
	}
	Import.AddCommand(initCustomiseMyData(cfg))

	CreateRepo = &cobra.Command{
		Use:   "create-repo",
		Short: "Creates a new repository with the typical Digital Publishing configurations ",
	}
	createGithubRepo := generateRepository()
	createGithubRepo.Flags().String("name", "dp-unnamed-application", "The name of the application, if "+
		"Digital specific application it should be prepended with 'dp-'")
	CreateRepo.AddCommand(createGithubRepo)

	Root = &cobra.Command{
		Use:   "dp-utils",
		Short: "dp-utils provides util functions for developers in ONS Digital Publishing",
	}
	GenerateApp := generateApplication()
	GenerateApp.Flags().String("name", "unset", "The name of the application, if "+
		"Digital specific application it should be prepended with 'dp-'")
	GenerateApp.Flags().String("go-version", "unset", "The version of Go the application should use")
	GenerateApp.Flags().String("project-location", "unset", "Location to generate project in")
	GenerateApp.Flags().String("create-repository", "n", "Should a repository be created for the "+
		"project, default no. Value can be y/Y/yes/YES/ or n/N/no/NO")
	GenerateApp.Flags().String("type", "unset", "Type of application to generate, values can "+
		"be: 'generic-program', 'base-application', 'api', 'controller', 'event-driven'")
	Root.AddCommand(Version, Clean, Import, CreateRepo, GenerateApp)

	return Root
}

// tearDownCustomiseMyData clean out all data from your local CMD stack.
func tearDownCustomiseMyData(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "cmd",
		Short: "Drop all CMD data from your local environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			err = zebedee.DeleteCollections()
			if err != nil {
				return err
			}

			err = customisemydata.DropMongoData(cfg)
			if err != nil {
				return err
			}

			err = customisemydata.DropNeo4jData(cfg)
			if err != nil {
				return err
			}

			return nil
		},
	}
}

// clearCollections delete all collections from your local publishing stack
func clearCollections() *cobra.Command {
	return &cobra.Command{
		Use:   "collections",
		Short: "Delete all Zebedee collections in your local environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			return zebedee.DeleteCollections()
		},
	}
}

// initCustomiseMyData import the prerequisite CMD data into your Mongo/Neo4j databases
func initCustomiseMyData(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "cmd",
		Short: "Import the prerequisite codelists and generic hierarchy data into your CMD environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			err = customisemydata.ImportGenericHierarchies(hierarchyBuilderPath, cfg)
			if err != nil {
				return err
			}

			err = customisemydata.ImportCodeLists(codeListScriptsPath, cfg)
			if err != nil {
				return err
			}

			return nil
		},
	}
}

func generateRepository() *cobra.Command {
	return &cobra.Command{
		Use:   "github",
		Short: "Creates a github hosted repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			nameOfApp, _ := cmd.Flags().GetString("name")
			err = repocreation.GenerateGithubRepository(nameOfApp)
			if err != nil {
				return err
			}

			return nil
		},
	}
}

func generateApplication() *cobra.Command {
	return &cobra.Command{
		Use:   "generate-application",
		Short: "Creates a github hosted repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			nameOfApp, _ := cmd.Flags().GetString("name")
			goVer, _ := cmd.Flags().GetString("go")
			projectLocation, _ := cmd.Flags().GetString("project-location")
			projType, _ := cmd.Flags().GetString("type")
			createRepository, _ := cmd.Flags().GetString("create-repository")

			err = appgen.GenerateApp(nameOfApp, projType, projectLocation, goVer)
			if err != nil {
				return err
			}
			createRepository = strings.ToLower(strings.TrimSpace(createRepository))
			if createRepository == "y" || createRepository == "yes" {
				repocreation.GenerateGithubRepository(nameOfApp)
			}

			return nil
		},
	}
}

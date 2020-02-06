package cmd

import (
	"context"
	"dp-cli/config"
	"dp-cli/customisemydata"
	"dp-cli/out"
	projectgeneration "dp-cli/project-generation"
	repository "dp-cli/repository-creation"
	"dp-cli/zebedee"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ONSdigital/log.go/log"

	"github.com/spf13/cobra"
)

var (
	root       *cobra.Command
	version    *cobra.Command
	clean      *cobra.Command
	importData *cobra.Command
	createRepo *cobra.Command

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

	version = &cobra.Command{
		Use:   "version",
		Short: "Print the app version",
		Run: func(cmd *cobra.Command, args []string) {
			out.Info(appVersion)
		},
	}

	clean = &cobra.Command{
		Use:   "clean",
		Short: "Clean/Delete data from your local environment",
	}
	clean.AddCommand(tearDownCustomiseMyData(cfg), clearCollections())

	importData = &cobra.Command{
		Use:   "import",
		Short: "ImportData your local developer environment",
	}
	importData.AddCommand(initCustomiseMyData(cfg))

	createRepo = &cobra.Command{
		Use:   "create-repo",
		Short: "Creates a new repository with the typical Digital Publishing configurations ",
	}
	createGithubRepo := generateRepository()
	createGithubRepo.Flags().String("name", "", "The name of the application, if "+
		"Digital specific application it should be prepended with 'dp-'")
	createGithubRepo.Flags().String("token", "", "The users personal access token")
	createGithubRepo.Flags().String("strategy", "git", "which branching-strategy this is depended on; will configure branches. Currently supported 'git' and 'github'")
	createRepo.AddCommand(createGithubRepo)

	root = &cobra.Command{
		Use:   "dp-cli",
		Short: "dp-cli provides util functions for developers in ONS Digital Publishing",
	}
	GenerateProject := generateApplication()
	GenerateProject.Flags().String("name", "", "The name of the application, if "+
		"Digital specific application it should be prepended with 'dp-'")
	GenerateProject.Flags().String("go-version", "", "The version of Go the application should use")
	GenerateProject.Flags().String("project-location", "", "Location to generate project in")
	GenerateProject.Flags().String("create-repository", "n", "Should a repository be created for the "+
		"project, default no. Value can be y/Y/yes/YES/ or n/N/no/NO")
	GenerateProject.Flags().String("type", "", "Type of application to generate, values can "+
		"be: 'generic-project', 'base-application', 'api', 'controller', 'event-driven'")
	GenerateProject.Flags().String("port", "", "The port this application will run on")
	root.AddCommand(version, clean, importData, createRepo, GenerateProject)

	return root
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
		RunE:  repository.RunGenerateRepo,
	}
}

func generateApplication() *cobra.Command {
	return &cobra.Command{
		Use:   "generate-project",
		Short: "Generates the boilerplate for a given project type",
		RunE:  RunGenerateApplication,
	}
}

func RunGenerateApplication(cmd *cobra.Command, args []string) error {
	var err error
	var cloneUrl string
	ctx := context.Background()
	nameOfApp, _ := cmd.Flags().GetString("name")
	goVer, _ := cmd.Flags().GetString("go")
	port, _ := cmd.Flags().GetString("port")
	projectLocation, _ := cmd.Flags().GetString("project-location")
	projectType, _ := cmd.Flags().GetString("type")
	createRepositoryInput, _ := cmd.Flags().GetString("create-repository")
	createRepositoryInput = strings.ToLower(strings.TrimSpace(createRepositoryInput))

	// Can't create repo unless project type has been provided in a flag, so prompt user for it
	if createRepositoryInput == "y" || createRepositoryInput == "yes" {

		listOfArguments := make(projectgeneration.ListOfArguments)

		listOfArguments["appName"] = &projectgeneration.Argument{
			InputVal:  nameOfApp,
			Context:   ctx,
			Validator: projectgeneration.ValidateAppName,
		}
		listOfArguments["projectType"] = &projectgeneration.Argument{
			InputVal:  projectType,
			Context:   ctx,
			Validator: projectgeneration.ValidateProjectType,
		}
		listOfArguments["projectLocation"] = &projectgeneration.Argument{
			InputVal:  projectLocation,
			Context:   ctx,
			Validator: projectgeneration.ValidateProjectLocation,
		}
		listOfArguments, err = projectgeneration.ValidateArguments(listOfArguments)
		if err != nil {
			log.Event(ctx, "input validation error", log.Error(err))
			return err
		}

		err := projectgeneration.ValidateProjectDirectory(ctx, listOfArguments["projectLocation"].OutputVal, listOfArguments["appName"].OutputVal)
		if err != nil {
			log.Event(ctx, "error confirming project directory is valid", log.Error(err))
			return err
		}

		prompt := "Please pick the branching strategy you wish this repo to use:"
		options := []string{"github flow", "git flow"}
		strategy, err := projectgeneration.OptionPromptInput(ctx, prompt, options...)

		if err != nil {
			log.Event(ctx, "error getting branch strategy", log.Error(err))
		}
		strategy = strings.Replace(strategy, " flow", "", -1)

		cloneUrl, err = repository.GenerateGithub(listOfArguments["appName"].OutputVal, projectgeneration.ProjectType(listOfArguments["projectType"].OutputVal), "", strategy)
		if err != nil {
			log.Event(ctx, "failed to generate project on github", log.Error(err))
			return err
		}
		err = repository.CloneRepository(ctx, cloneUrl, listOfArguments["projectLocation"].OutputVal, listOfArguments["appName"].OutputVal)
		if err != nil {
			log.Event(ctx, "failed to clone repository", log.Error(err))
			return err
		}
		err = projectgeneration.GenerateProject(listOfArguments["appName"].OutputVal, listOfArguments["projectType"].OutputVal, listOfArguments["projectLocation"].OutputVal, goVer, port, true)
		if err != nil {
			log.Event(ctx, "failed to generate project on github", log.Error(err))
			return err
		}
		err = repository.PushToRepo(ctx, listOfArguments["projectLocation"].OutputVal, listOfArguments["appName"].OutputVal)
		if err != nil {
			log.Event(ctx, "failed to push to repository", log.Error(err))
			return err
		}
		return nil
	}
	err = projectgeneration.GenerateProject(nameOfApp, projectType, projectLocation, goVer, port, false)
	if err != nil {
		return err
	}
	return nil
}

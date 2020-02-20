package cmd

import (
	"context"
	projectgeneration "dp-cli/project-generation"
	"dp-cli/repository-creation"
	"strings"

	"github.com/ONSdigital/log.go/log"
	"github.com/spf13/cobra"
)

func generateProjectSubCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "generate-project",
		Short: "Generates the boilerplate for a given project type",
		RunE:  RunGenerateApplication,
	}

	command.Flags().String("name", "", "The name of the application, if Digital specific application it should be prepended with 'dp-'")
	command.Flags().String("go-version", "", "The version of Go the application should use")
	command.Flags().String("project-location", "", "Location to generate project in")
	command.Flags().String("create-repository", "n", "Should a repository be created for the project, default no. Value can be y/Y/yes/YES/ or n/N/no/NO")
	command.Flags().String("type", "", "Type of application to generate, values can be: 'generic-project', 'base-application', 'api', 'controller', 'event-driven'")
	command.Flags().String("port", "", "The port this application will run on")

	return command
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

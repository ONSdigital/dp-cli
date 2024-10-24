package command

import (
	"context"
	"strings"

	"github.com/ONSdigital/dp-cli/project_generation"
	"github.com/ONSdigital/dp-cli/repository_creation"

	"github.com/ONSdigital/log.go/v2/log"
	"github.com/spf13/cobra"
)

func generateProjectSubCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "generate-project",
		Short: "Generates the boilerplate for a given project type",
		RunE:  RunGenerateApplication,
	}

	command.Flags().String("name", "", "The name of the application, if Digital specific application it should be prepended with 'dp-'")
	command.Flags().String("description", "", "A short sentence to describe the application")
	command.Flags().String("go-version", "", "The version of Go the application should use")
	command.Flags().String("project-location", "", "Location to generate project in")
	command.Flags().String("create-repository", "n", "Should a repository be created for the project, default no. Value can be y/Y/yes/YES/ or n/N/no/NO")
	command.Flags().String("type", "", "Type of application to generate, values can be: 'generic-project', 'base-application', 'api', 'controller', 'event-driven', 'library'")
	command.Flags().String("port", "", "The port this application will run on")
	command.Flags().String("strategy", "git", "which branching-strategy this is depended on; will configure branches. Currently supported 'git' and 'github'")
	command.Flags().String("team-slugs", "", "Comma separated list of teams, e.g. 'dissemination-open-sauce', who are the codeowners of this project")

	return command
}

// RunGenerateApplication will create and setup the repo
func RunGenerateApplication(cmd *cobra.Command, args []string) (err error) {
	cloneURL := ""
	ctx := context.Background()
	nameOfApp, _ := cmd.Flags().GetString("name")
	appDescription, _ := cmd.Flags().GetString("description")
	goVer, _ := cmd.Flags().GetString("go-version")
	port, _ := cmd.Flags().GetString("port")
	projectLocation, _ := cmd.Flags().GetString("project-location")
	projectType, _ := cmd.Flags().GetString("type")
	createRepositoryInput, _ := cmd.Flags().GetString("create-repository")
	createRepositoryInput = strings.ToLower(strings.TrimSpace(createRepositoryInput))
	strategy, _ := cmd.Flags().GetString("strategy")
	teamSlugs, _ := cmd.Flags().GetString("team-slugs")

	// Can't create repo unless project type has been provided in a flag, so prompt user for it
	if createRepositoryInput == "y" || createRepositoryInput == "yes" {

		listOfArguments := make(project_generation.ListOfArguments)

		listOfArguments["appName"] = &project_generation.Argument{
			InputVal:  nameOfApp,
			Context:   ctx,
			Validator: project_generation.ValidateAppName,
		}
		listOfArguments["description"] = &project_generation.Argument{
			InputVal:  appDescription,
			Context:   ctx,
			Validator: project_generation.ValidateAppDescription,
		}
		listOfArguments["projectType"] = &project_generation.Argument{
			InputVal:  projectType,
			Context:   ctx,
			Validator: project_generation.ValidateProjectType,
		}
		listOfArguments["projectLocation"] = &project_generation.Argument{
			InputVal:  projectLocation,
			Context:   ctx,
			Validator: project_generation.ValidateProjectLocation,
		}
		listOfArguments["strategy"] = &project_generation.Argument{
			InputVal:  strategy,
			Context:   ctx,
			Validator: project_generation.ValidateBranchingStrategy,
		}
		listOfArguments, err = project_generation.ValidateArguments(listOfArguments)
		if err != nil {
			log.Error(ctx, "input validation error", err)
			return err
		}

		err := project_generation.ValidateProjectDirectory(ctx, listOfArguments["projectLocation"].OutputVal, listOfArguments["appName"].OutputVal)
		if err != nil {
			log.Error(ctx, "error confirming project directory is valid", err)
			return err
		}
		cloneURL, err = repository_creation.GenerateGithub(listOfArguments["appName"].OutputVal, listOfArguments["description"].OutputVal, project_generation.ProjectType(listOfArguments["projectType"].OutputVal), "", listOfArguments["strategy"].OutputVal, listOfArguments["teamSlugs"].OutputVal)
		if err != nil {
			log.Error(ctx, "failed to generate project on github", err)
			return err
		}
		err = repository_creation.CloneRepository(ctx, cloneURL, listOfArguments["projectLocation"].OutputVal, listOfArguments["appName"].OutputVal)
		if err != nil {
			log.Error(ctx, "failed to clone repository", err)
			return err
		}
		err = project_generation.GenerateProject(listOfArguments["appName"].OutputVal, listOfArguments["description"].OutputVal, listOfArguments["projectType"].OutputVal, listOfArguments["projectLocation"].OutputVal, goVer, port, listOfArguments["teamSlugs"].OutputVal, true)
		if err != nil {
			log.Error(ctx, "failed to generate project on github", err)
			return err
		}
		err = repository_creation.PushToRepo(ctx, listOfArguments["projectLocation"].OutputVal, listOfArguments["appName"].OutputVal)
		if err != nil {
			log.Error(ctx, "failed to push to repository", err)
			return err
		}
		return nil
	}
	err = project_generation.GenerateProject(nameOfApp, appDescription, projectType, projectLocation, goVer, port, teamSlugs, false)
	if err != nil {
		return err
	}
	return nil
}

package cmd

import (
	"github.com/ONSdigital/dp-cli/repository_creation"

	"github.com/spf13/cobra"
)

func createRepoSubCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "create-repo",
		Short: "Creates a new repository with the typical Digital Publishing configurations ",
	}

	command.AddCommand(createGithubRepo())
	return command
}

func createGithubRepo() *cobra.Command {
	command := &cobra.Command{
		Use:   "github",
		Short: "Creates a github hosted repository",
		RunE:  repository_creation.RunGenerateRepo,
	}

	command.Flags().String("name", "", "The name of the application, if Digital specific application it should be prepended with 'dp-'")
	command.Flags().String("token", "", "The users personal access token")
	command.Flags().String("strategy", "git", "which branching-strategy this is depended on; will configure branches. Currently supported 'git' and 'github'")
	return command
}


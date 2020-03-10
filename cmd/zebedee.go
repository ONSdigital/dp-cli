package cmd

import (
	"github.com/ONSdigital/dp-cli/zebedee"
	"github.com/spf13/cobra"
)

var dir string

func zebedeeCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "zebedee",
		Short: "Zebedee CMS actions",
	}

	c.AddCommand(generateCommand())
	return c
}

func generateCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "generate",
		Short: "Generate Zebedee stuff",
	}

	c.AddCommand(contentCommand())
	return c
}

func contentCommand() *cobra.Command {

	c := &cobra.Command{
		Use:   "content",
		Short: "Create a new Zebedee CMS file system structure populated with the default content, teams, users etc.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return zebedee.NewCMS()
		},
	}

	//c.Flags().StringVarP(&dir, "content_dir", "d", "", "The path of the directory to create the content in")

	return c
}

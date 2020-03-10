package cmd

import (
	"github.com/ONSdigital/dp-cli/out"
	"github.com/spf13/cobra"
)

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
			out.WriteF(out.INFO, "generating new zebedee content")
			return nil
		},
	}
	return c
}

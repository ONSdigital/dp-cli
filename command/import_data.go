package command

import (
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/customisemydata"

	"github.com/spf13/cobra"
)

func importDataSubCommand(cfg *config.Config) *cobra.Command {
	command := &cobra.Command{
		Use:   "import",
		Short: "Import data into your local developer environment",
	}

	command.AddCommand(&cobra.Command{
		Use:   "cmd",
		Short: "Import the prerequisite codelists and generic hierarchy data into your CMD environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			err = customisemydata.ImportGenericHierarchies(cfg)
			if err != nil {
				return err
			}

			err = customisemydata.ImportCodeLists(cfg)
			if err != nil {
				return err
			}

			return nil
		},
	})

	return command
}

// initCustomiseMyData import the prerequisite CMD data into your Mongo/Neo4j databases
func initCustomiseMyData(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "cmd",
		Short: "Import the prerequisite codelists and generic hierarchy data into your CMD environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			err = customisemydata.ImportGenericHierarchies(cfg)
			if err != nil {
				return err
			}

			err = customisemydata.ImportCodeLists(cfg)
			if err != nil {
				return err
			}

			return nil
		},
	}
}

package cmd

import (
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/customisemydata"
	"github.com/ONSdigital/dp-cli/zebedee"

	"github.com/spf13/cobra"
)

func cleanSubCommand(cfg *config.Config) *cobra.Command {
	command := &cobra.Command{
		Use:   "clean",
		Short: "Clean/Delete data from your local environment",
	}
	command.AddCommand(tearDownCustomiseMyData(cfg), clearCollections())
	return command
}

// tearDownCustomiseMyData is a child command of clean that cleans out data from your local CMD stack.
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

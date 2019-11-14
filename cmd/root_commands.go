package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Root = &cobra.Command{
		Use:   "dp-utils",
		Short: "dp-utils provides util functions for developers in ONS Digital Publishing",
	}

	Version = &cobra.Command{
		Use:   "version",
		Short: "Print the app version",
		Long:  "Print the app version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("dp-utils v0.0.1")
		},
	}
)

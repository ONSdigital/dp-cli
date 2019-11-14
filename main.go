package main

import (
	"dp-utils/cmd"
	"dp-utils/config"
	"os"
)

func main() {
	cfg, _ := config.Get()
	root := cmd.Root

	root.AddCommand(cmd.Version, cmd.Cleaning(cfg), cmd.Initialise(cfg))
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

package main

import (
	"dp-cli/cmd"
	"dp-cli/config"
	"dp-cli/out"
	"os"
)

func main() {
	cfg, err := config.Get()
	if err != nil {
		out.Error(err)
		os.Exit(1)
	}

	root := cmd.Load(cfg)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

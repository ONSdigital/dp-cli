package main

import (
	"dp-utils/cmd"
	"dp-utils/config"
	"dp-utils/out"
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

package main

import (
	"dp-cli/cmd"
	"dp-cli/config"
	"dp-cli/out"
	"os"
)

func main() {
	if err := run(os.Args); err != nil {
		out.Error(err)
		os.Exit(1)
	}
}

// run the dp-cli application
func run(args []string) error {
	cfg, err := config.Get()
	if err != nil {
		return err
	}

	root := cmd.Load(cfg)

	return root.Execute()
}

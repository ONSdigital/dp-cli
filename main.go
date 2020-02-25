package main

import (
	"os"

	"github.com/ONSdigital/dp-cli/cmd"
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"
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

	root, err := cmd.Load(cfg)
	if err != nil {
		return err
	}

	return root.Execute()
}

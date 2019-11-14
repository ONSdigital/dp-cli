package cmd

import (
	"dp-utils/config"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	goPath               = os.Getenv("GOPATH")
	onsDigitalPath       = filepath.Join(goPath, "src/github.com/ONSdigital")
	hierarchyBuilderPath = filepath.Join(onsDigitalPath, "dp-hierarchy-builder/cypher-scripts")
	codeListScriptsPath  = filepath.Join(onsDigitalPath, "dp-code-list-scripts/code-list-scripts")
)

func Initialise(cfg *config.Config) *cobra.Command {
	c := &cobra.Command{
		Use:   "init",
		Short: "Initialise your local developer environment",
	}
	c.AddCommand(initCustomiseMyData(cfg))
	return c
}

func initCustomiseMyData(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "cmd",
		Short: "Import the prerequisite codelists and generic hierarchy data into your CMD environment",
		Run: func(cmd *cobra.Command, args []string) {
			importGenericHierarchies(cfg)
			importCodelists(cfg)
		},
	}
}

func importGenericHierarchies(cfg *config.Config) error {
	if len(cfg.CMD.Hierarchies) == 0 {
		output("No hierarchies defined in config skipping step")
		return nil
	}

	output(fmt.Sprintf("Building generic hierarchies: %+v", cfg.CMD.Hierarchies))

	for _, script := range cfg.CMD.Hierarchies {
		command := fmt.Sprintf("cypher-shell < %s/%s", hierarchyBuilderPath, script)

		if err := execCommand(command, ""); err != nil {
			return err
		}
	}

	output("Hierarchies built successfully")
	return nil
}

func importCodelists(cfg *config.Config) error {
	if len(cfg.CMD.Codelists) == 0 {
		output("No code lists defined in config skipping step")
		return nil
	}

	output(fmt.Sprintf("Importing code lists: %+v", cfg.CMD.Codelists))

	for _, codelist := range cfg.CMD.Codelists {
		command := fmt.Sprintf("./load -q=%s -f=%s", "cypher", codelist)

		if err := execCommand(command, codeListScriptsPath); err != nil {
			return err
		}
	}

	output("Code lists imported successfully")
	return nil
}

func execCommand(command string, wrkDir string) error {
	cmd := exec.Command("bash", "-c", command)
	cmd.Stderr = os.Stderr

	if len(wrkDir) > 0 {
		cmd.Dir = wrkDir
	}

	return cmd.Run()
}

func output(msg string) {
	color.Magenta(fmt.Sprintf("[import] %s", msg))
}

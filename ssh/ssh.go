package ssh

import (
	"fmt"

	"github.com/ONSdigital/dp-cli/ansible"
	"github.com/ONSdigital/dp-cli/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type RunFunc func(cmd *cobra.Command, args []string) error

func Command_(cfg *config.Config) (*cobra.Command, error) {
	c := &cobra.Command{
		Use:   "ssh",
		Short: "access an environment using ssh",
	}

	var subcommands []*cobra.Command
	for _, env := range cfg.SSHConfig.Environments {

		sub, err := createEnvSubCommand(cfg.DPSetupPath, env)
		if err != nil {
			return nil, errors.WithMessagef(err, "error creating subcommand for env: %s", env.Name)
		}
		subcommands = append(subcommands, sub)
	}

	c.AddCommand(subcommands...)
	return c, nil
}

func createEnvSubCommand(dpSetupPath string, env config.Environment) (*cobra.Command, error) {
	sub := &cobra.Command{
		Use:   env.Name,
		Short: "ssh to " + env.Name,
	}

	_, err := ansible.GetGroupsForEnvironment(dpSetupPath, env.Name)
	if err != nil {
		return nil, errors.WithMessagef(err, "error loading ansible hosts for %s\n", env.Name)
	}

	sub.RunE = func(cmd *cobra.Command, args []string) error {
		fmt.Println("ssh-ing to " + env.Name)
		return nil
	}

	return sub, nil
}

func newRunFunc(cfg *config.Config) RunFunc {
	return func(cmd *cobra.Command, args []string) error {
		if len(cfg.SSHConfig.User) == 0 {
			return errors.New("DP_SSH_USER environment variable must be set")
		}
		return nil
	}
}

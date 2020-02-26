package ssh

import (
	"github.com/ONSdigital/dp-cli/ansible"
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type runFunc func(cmd *cobra.Command, args []string) error

func Command(cfg *config.Config) (*cobra.Command, error) {
	c := &cobra.Command{
		Use:   "ssh",
		Short: "access an environment using ssh",
	}

	var subCommands []*cobra.Command
	for _, env := range cfg.SSHConfig.Environments {

		sub, err := createEnvSubCommand(cfg, env)
		if err != nil {
			return nil, errors.WithMessagef(err, "error creating sub-command for env: %s", env.Name)
		}
		subCommands = append(subCommands, sub)
	}

	c.AddCommand(subCommands...)
	return c, nil
}

func createEnvSubCommand(cfg *config.Config, env config.Environment) (*cobra.Command, error) {
	sub := &cobra.Command{
		Use:   env.Name,
		Short: "ssh to " + env.Name,
	}

	_, err := ansible.GetGroupsForEnvironment(cfg.DPSetupPath, env.Name)
	if err != nil {
		return nil, errors.WithMessagef(err, "error loading ansible hosts for %s\n", env.Name)
	}

	sub.RunE = newRunFunc(cfg, env)

	return sub, nil
}

func newRunFunc(cfg *config.Config, env config.Environment) runFunc {
	return func(cmd *cobra.Command, args []string) error {
		if len(cfg.SSHConfig.User) == 0 {
			return errors.New("DP_SSH_USER environment variable must be set")
		}

		out.InfoFHighlight("ssh to %s end", env.Name)
		return nil
	}
}

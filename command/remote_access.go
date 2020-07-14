package command

import (
	"github.com/ONSdigital/dp-cli/aws"
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"
	"github.com/spf13/cobra"
)

var specialEnvs = []string{"concourse"}

func remoteAccess(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remote",
		Short: "Allow or deny remote access to environment",
	}

	subCommands := []*cobra.Command{
		allowCommand(cfg.SSHUser, cfg.Environments),
		denyCommand(cfg.SSHUser, cfg.Environments),
	}

	cmd.AddCommand(subCommands...)
	return cmd
}

// build the allow sub command - has a sub commands for each environment.
func allowCommand(sshUser string, envs []config.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "allow",
		Short: "allow access to environment",
	}

	cmds := make([]*cobra.Command, 0)

	for _, e := range envs {
		env := e
		cmds = append(cmds, &cobra.Command{
			Use:   e.Name,
			Short: "allow access to " + env.Name,
			RunE: func(cmd *cobra.Command, args []string) error {
				lvl := out.GetLevel(env)
				out.Highlight(lvl, "allowing access to %s", env.Name)
				return aws.AllowIPForEnvironment(sshUser, env.Name, env.Profile)
			},
		})
	}

	for _, e := range specialEnvs {
		cmds = append(cmds, &cobra.Command{
			Use:   e,
			Short: "allow access to " + e,
			RunE: func(cmd *cobra.Command, args []string) error {
				out.Highlight(out.INFO, "allowing access to %s", e)
				return aws.AllowIPForEnvironment(sshUser, e, "")
			},
		})
	}

	c.AddCommand(cmds...)
	return c
}

// build the deny sub command - has a sub command for each environment
func denyCommand(sshUser string, envs []config.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "deny",
		Short: "deny access to environment",
	}

	cmds := make([]*cobra.Command, 0)

	for _, e := range envs {
		env := e
		cmds = append(cmds, &cobra.Command{
			Use:   e.Name,
			Short: "deny access to " + env.Name,
			RunE: func(cmd *cobra.Command, args []string) error {
				lvl := out.GetLevel(env)
				out.Highlight(lvl, "denying access to %s", env.Name)
				return aws.DenyIPForEnvironment(sshUser, env.Name, env.Profile)
			},
		})
	}

	for _, e := range specialEnvs {
		cmds = append(cmds, &cobra.Command{
			Use:   e,
			Short: "deny access to " + e,
			RunE: func(cmd *cobra.Command, args []string) error {
				out.Highlight(out.INFO, "denying access to %s", e)
				return aws.DenyIPForEnvironment(sshUser, e, "")
			},
		})
	}

	c.AddCommand(cmds...)
	return c
}

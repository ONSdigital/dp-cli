package cmd

import (
	"github.com/ONSdigital/dp-cli/aws"
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"
	"github.com/spf13/cobra"
)

func remoteAccess(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remote",
		Short: "Allow or deny remote access to environment",
	}

	subCommands := []*cobra.Command{
		allowCommand(cfg.SSHUser, cfg.Environments),
		denyCommand(cfg.SSHUser, cfg.Environments),
		concourseCommand(cfg.SSHUser),
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

	c.AddCommand(cmds...)
	return c
}

// build the concourse sub command - has allow and deny sub commands.
func concourseCommand(sshUser string) *cobra.Command {
	c := &cobra.Command{
		Use:   "concourse",
		Short: "allow or deny access to concourse",
	}

	allow := &cobra.Command{
		Use:   "allow",
		Short: "allow access to concourse",
		RunE: func(cmd *cobra.Command, args []string) error {
			out.WriteF(out.INFO, "allow access to concourse")
			return aws.AllowIPForConcourse(sshUser)
		},
	}

	deny := &cobra.Command{
		Use:   "deny",
		Short: "deny access to concourse",
		RunE: func(cmd *cobra.Command, args []string) error {
			out.WriteF(out.INFO, "denying access to concourse")
			return aws.DenyIPForConcourse(sshUser)
		},
	}

	c.AddCommand(allow, deny)
	return c
}

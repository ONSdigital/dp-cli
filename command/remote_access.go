package command

import (
	"github.com/ONSdigital/dp-cli/aws"
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"
	"github.com/spf13/cobra"
)

var specialEnvs = []config.Environment{{
	Name:    "concourse",
	Profile: "",
}}

func remoteAccess(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remote",
		Short: "Allow or deny remote access to environment",
	}
	opts := config.WithOpts{
		ForUser: cmd.PersistentFlags().StringP("ssh-user", "u", cfg.SSHUser, "use <arg> as ssh user"),
	}

	subCommands := []*cobra.Command{
		allowCommand(cfg.Environments, opts),
		denyCommand(cfg.Environments, opts),
	}

	cmd.AddCommand(subCommands...)
	return cmd
}

// build the allow sub command - has a sub commands for each environment.
func allowCommand(envs []config.Environment, opts config.WithOpts) *cobra.Command {
	c := &cobra.Command{
		Use:   "allow",
		Short: "allow access to environment",
	}

	skipDeny := c.PersistentFlags().BoolP("no-deny", "D", false, "Skip any 'deny' of existing IPs - allows >1 IP for user")
	opts.HTTPOnly = c.PersistentFlags().BoolP("http-only", "H", false, "Add only HTTP ports - no ssh")

	cmds := make([]*cobra.Command, 0)

	for _, e := range append(envs, specialEnvs...) {
		env := e
		cmds = append(cmds, &cobra.Command{
			Use:   e.Name,
			Short: "allow access to " + env.Name,
			RunE: func(cmd *cobra.Command, args []string) error {
				lvl := out.GetLevel(env)
				if !*skipDeny {
					out.Highlight(lvl, "removing existing access to %s", env.Name)
					aws.DenyIPForEnvironment(env.Name, env.Profile, env.ExtraPorts, opts)
				}
				out.Highlight(lvl, "allowing access to %s", env.Name)
				return aws.AllowIPForEnvironment(env.Name, env.Profile, env.ExtraPorts, opts)
			},
		})
	}

	c.AddCommand(cmds...)
	return c
}

// build the deny sub command - has a sub command for each environment
func denyCommand(envs []config.Environment, opts config.WithOpts) *cobra.Command {
	c := &cobra.Command{
		Use:   "deny",
		Short: "deny access to environment",
	}

	cmds := make([]*cobra.Command, 0)

	for _, e := range append(envs, specialEnvs...) {
		env := e
		cmds = append(cmds, &cobra.Command{
			Use:   e.Name,
			Short: "deny access to " + env.Name,
			RunE: func(cmd *cobra.Command, args []string) error {
				lvl := out.GetLevel(env)
				out.Highlight(lvl, "denying access to %s", env.Name)
				return aws.DenyIPForEnvironment(env.Name, env.Profile, env.ExtraPorts, opts)
			},
		})
	}

	c.AddCommand(cmds...)
	return c
}

package command

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

	ipFlag := cmd.PersistentFlags().String("ip", "", "The IP for ssh,remote sub-commands")
	if ipFlag != nil {
		cfg.IPAddress = ipFlag
	}
	userDefault := ""
	if cfg.User != nil {
		userDefault = *cfg.User
	}
	userFlag := cmd.PersistentFlags().String("user", userDefault, "The user for access lists")
	if userFlag != nil {
		cfg.User = userFlag
	}

	subCommands := []*cobra.Command{
		allowCommand(cfg.User, cfg.Environments, cfg),
		denyCommand(cfg.User, cfg.Environments, cfg),
	}

	cmd.AddCommand(subCommands...)
	return cmd
}

// build the allow sub command - has a sub commands for each environment.
func allowCommand(sshUser *string, envs []config.Environment, cfg *config.Config) *cobra.Command {
	c := &cobra.Command{
		Use:   "allow",
		Short: "allow access to environment",
	}

	skipDeny := c.PersistentFlags().BoolP("no-deny", "D", false, "Skip any 'deny' of existing IPs - allows >1 IP for user")
	cfg.HttpOnly = c.PersistentFlags().BoolP("http-only", "H", false, "Allow only http-related ports (no ssh)")

	cmds := make([]*cobra.Command, 0)

	for _, e := range envs {
		env := e
		cmds = append(cmds, &cobra.Command{
			Use:   e.Name,
			Short: "allow access to " + env.Name,
			RunE: func(cmd *cobra.Command, args []string) error {
				lvl := out.GetLevel(env)
				if !*skipDeny {
					out.Highlight(lvl, "removing existing access to %s", env.Name)
					aws.DenyIPForEnvironment(sshUser, env.Name, env.Profile, env.ExtraPorts, cfg)
				}
				out.Highlight(lvl, "allowing access to %s", env.Name)
				return aws.AllowIPForEnvironment(sshUser, env.Name, env.Profile, env.ExtraPorts, cfg)
			},
		})
	}

	c.AddCommand(cmds...)
	return c
}

// build the deny sub command - has a sub command for each environment
func denyCommand(sshUser *string, envs []config.Environment, cfg *config.Config) *cobra.Command {
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
				return aws.DenyIPForEnvironment(sshUser, env.Name, env.Profile, env.ExtraPorts, cfg)
			},
		})
	}

	c.AddCommand(cmds...)
	return c
}

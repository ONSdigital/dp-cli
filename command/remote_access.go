package command

import (
	"github.com/ONSdigital/dp-cli/aws"
	"github.com/ONSdigital/dp-cli/cli"
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
	if cfg.UserName != nil {
		userDefault = *cfg.UserName
	}
	userFlag := cmd.PersistentFlags().String("user", userDefault, "The user for access lists")
	if userFlag != nil {
		cfg.UserName = userFlag
	}

	subCommands := []*cobra.Command{
		allowCommand(cfg.UserName, cfg.Environments, cfg),
		denyCommand(cfg.UserName, cfg.Environments, cfg),
		loginCommand(cfg.UserName, cfg.Environments, cfg),
	}

	cmd.AddCommand(subCommands...)
	return cmd
}

// build the allow sub command - has a sub commands for each environment.
func allowCommand(userName *string, envs []config.Environment, cfg *config.Config) *cobra.Command {
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
					aws.DenyIPForEnvironment(userName, env.Name, cfg.GetProfile(env.Name), env.ExtraPorts, cfg)
				}
				out.Highlight(lvl, "allowing access to %s", env.Name)
				return aws.AllowIPForEnvironment(userName, env.Name, cfg.GetProfile(env.Name), env.ExtraPorts, cfg)
			},
		})
	}

	c.AddCommand(cmds...)
	return c
}

// build the deny sub command - has a sub command for each environment
func denyCommand(userName *string, envs []config.Environment, cfg *config.Config) *cobra.Command {
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
				return aws.DenyIPForEnvironment(userName, env.Name, cfg.GetProfile(env.Name), env.ExtraPorts, cfg)
			},
		})
	}

	c.AddCommand(cmds...)
	return c
}

// loginCommand - build the `login` sub-command - has a sub-command for each environment
func loginCommand(userName *string, envs []config.Environment, cfg *config.Config) *cobra.Command {
	firstEnv := envs[0]
	c := &cobra.Command{
		Use:   "login",
		Short: "login to AWS environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			lvl := out.GetLevel(firstEnv)
			loginCmd := "aws sso login --profile " + cfg.GetProfile(firstEnv.Name)
			out.Highlight(lvl, "logging in to %s using %s", firstEnv.Name, loginCmd)
			return cli.ExecCommand(loginCmd, ".")
		},
	}

	return c
}

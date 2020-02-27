package cmd

import (
	"fmt"
	"strconv"

	"github.com/ONSdigital/dp-cli/ansible"
	"github.com/ONSdigital/dp-cli/aws"
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/ssh"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Command builds an cobra.Command to SSH into an environment.
// The command has the following structure:
//
// 	ssh
// 	  environment
// 		group
//		  instance
//
func sshCommand(cfg *config.Config) (*cobra.Command, error) {
	sshC := &cobra.Command{
		Use:   "ssh",
		Short: "access an environment using ssh",
	}

	environmentCommands, err := createEnvironmentCommands(cfg)
	if err != nil {
		return nil, err
	}

	sshC.AddCommand(environmentCommands...)
	return sshC, nil
}

// create a array of ssh sub commands for the available environments
func createEnvironmentCommands(cfg *config.Config) ([]*cobra.Command, error) {
	commands := make([]*cobra.Command, 0)

	for _, env := range cfg.SSHConfig.Environments {
		envC := &cobra.Command{
			Use:   env.Name,
			Short: "ssh to " + env.Name,
		}

		groupCommands, err := createEnvironmentGroupCommands(env, cfg)
		if err != nil {
			return nil, errors.WithMessagef(err, "error creating group commands for env: %s", env.Name)
		}

		envC.AddCommand(groupCommands...)
		commands = append(commands, envC)
	}
	return commands, nil
}

// create a array of environment sub commands for each group available in the chosen environment
func createEnvironmentGroupCommands(env config.Environment, cfg *config.Config) ([]*cobra.Command, error) {
	groups, err := ansible.GetGroupsForEnvironment(cfg.DPSetupPath, env.Name)
	if err != nil {
		return nil, errors.WithMessagef(err, "error loading ansible hosts for %s\n", env.Name)
	}

	commands := make([]*cobra.Command, 0)

	for _, grp := range groups {
		instances, err := aws.ListEC2ByAnsibleGroup(env.Name, env.Profile, grp)
		if err != nil {
			return nil, errors.WithMessagef(err, "error fetching ec2: %q", env)
		}

		if len(instances) == 0 {
			// no instances available so skip creating a command
			continue
		}

		grpC := &cobra.Command{
			Use:   grp,
			Short: fmt.Sprintf("ssh to %s %s", env.Name, grp),
		}

		instanceCommands, err := createInstanceCommands(grp, cfg, env, instances)
		if err != nil {
			return nil, err
		}

		grpC.AddCommand(instanceCommands...)
		commands = append(commands, grpC)
	}

	return commands, nil
}

// create a array of group sub commands for each instance available in the chosen environment group
func createInstanceCommands(grp string, cfg *config.Config, env config.Environment, instances []aws.EC2Result) ([]*cobra.Command, error) {
	commands := make([]*cobra.Command, 0)

	for i, instance := range instances {
		index := strconv.Itoa(i + 1)

		instanceC := &cobra.Command{
			Use:   index,
			Short: fmt.Sprintf("ssh to %s %s", grp, instance.Name),
			RunE: func(cmd *cobra.Command, args []string) error {
				return ssh.Launch(cfg, env, instance)
			},
		}

		commands = append(commands, instanceC)
	}
	return commands, nil
}

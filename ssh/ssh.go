package ssh

import (
	"fmt"
	"strconv"

	"github.com/ONSdigital/dp-cli/ansible"
	"github.com/ONSdigital/dp-cli/aws"
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"
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
func Command(cfg *config.Config) (*cobra.Command, error) {
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

		instCount := len(instances)
		if instCount == 0 {
			continue
		}

		grpC := &cobra.Command{
			Use:   grp,
			Short: fmt.Sprintf("ssh to a %s instance in %s - %d available", grp, env.Name, instCount),
		}

		instanceCommands, err := createInstanceCommands(cfg.SSHConfig.User, grp, env, instances)
		if err != nil {
			return nil, err
		}

		grpC.AddCommand(instanceCommands...)
		commands = append(commands, grpC)
	}

	return commands, nil
}

// create a array of group sub commands for each instance available in the chosen environment group
func createInstanceCommands(shUser, grp string, env config.Environment, instances []aws.EC2Result) ([]*cobra.Command, error) {
	commands := make([]*cobra.Command, 0)

	for i, instance := range instances {

		index := strconv.Itoa(i + 1)
		instanceC := &cobra.Command{
			Use:   index,
			Short: fmt.Sprintf("ssh to %s instance %s", grp, instance.Name),
			Run: func(cmd *cobra.Command, args []string) {
				out.InfoFHighlight("env %s profile %s group %s instance %s", env.Name, env.Profile, grp, index)
			},
		}

		commands = append(commands, instanceC)
	}
	return commands, nil
}

func getLogger(env config.Environment) out.Log {
	logFunc := out.InfoFHighlight
	if env.Name == "production" {
		logFunc = out.ErrorFHighlight
	}
	return logFunc
}

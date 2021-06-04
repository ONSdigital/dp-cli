package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ONSdigital/dp-cli/ansible"
	"github.com/ONSdigital/dp-cli/aws"
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"
	"github.com/ONSdigital/dp-cli/ssh"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// sshCommand builds a cobra.Command to SSH into an environment.
// The command has the following structure:
//
// 	ssh
// 	    environment 	# develop
// 		group		# publishing_mount
//		    instance	# 1
//
func sshCommand(cfg *config.Config) (*cobra.Command, error) {
	sshC := &cobra.Command{
		Use:   "ssh",
		Short: "Access an environment using ssh",
	}

	portArgs := sshC.PersistentFlags().StringSliceP("port", "p", nil, "Optional port forwarding rule[s] of the form `[<local>:[<host>:]]<remote>` e.g. '15900', '8080:15900', '15900,8080:15900', '1234:hostX:4321'")
	verboseCount := sshC.PersistentFlags().CountP("verbose", "v", "verbose - increase ssh verbosity")
	environmentCommands, err := createEnvironmentSubCommands(cfg, portArgs, verboseCount)
	if err != nil {
		return nil, err
	}
	if len(environmentCommands) == 0 {
		out.Warn("Warning: No subcommands found for envs - missing envs in config?")
	}

	sshC.AddCommand(environmentCommands...)
	return sshC, nil
}

// create a array of environment sub commands available to ssh to.
func createEnvironmentSubCommands(cfg *config.Config, portArgs *[]string, verboseCount *int) ([]*cobra.Command, error) {
	commands := make([]*cobra.Command, 0)

	for _, env := range cfg.Environments {
		envC := &cobra.Command{
			Use:   env.Name,
			Short: "ssh to " + env.Name,
		}

		groupCommands, err := createEnvironmentGroupSubCommands(env, cfg, portArgs, verboseCount)
		if err != nil {
			return nil, errors.WithMessagef(err, "error creating group commands for env: %s", env.Name)
		}

		envC.AddCommand(groupCommands...)
		commands = append(commands, envC)
	}
	return commands, nil
}

// create a array of environment group sub commands available to ssh to.
func createEnvironmentGroupSubCommands(env config.Environment, cfg *config.Config, portArgs *[]string, verboseCount *int) ([]*cobra.Command, error) {
	groups, err := ansible.GetGroupsForEnvironment(cfg.DPSetupPath, env.Name)
	if err != nil {
		return nil, errors.WithMessagef(err, "error loading ansible hosts for %s", env.Name)
	}

	commands := make([]*cobra.Command, 0)
	seenIP := make(map[string]bool)

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

		instanceCommands, err := createInstanceSubCommands(grp, cfg, env, instances, portArgs, verboseCount)
		if err != nil {
			return nil, err
		}

		grpC.AddCommand(instanceCommands...)
		commands = append(commands, grpC)

		for _, inst := range instances {
			if _, ok := seenIP[inst.IPAddress]; ok {
				continue
			}
			seenIP[inst.IPAddress] = true

			e := env
			instX := inst
			ipC := &cobra.Command{
				Use:   inst.IPAddress,
				Short: fmt.Sprintf("ssh to %s %-15s [%s]", env.Name, inst.IPAddress, strings.Join(inst.GroupAKA, ", ")),
				RunE: func(cmd *cobra.Command, args []string) error {
					return ssh.Launch(cfg, e, instX, portArgs, verboseCount, args)
				},
			}
			commands = append(commands, ipC)
		}
	}

	return commands, nil
}

// create a array of instance sub commands available to ssh to.
func createInstanceSubCommands(grp string, cfg *config.Config, env config.Environment, instances []aws.EC2Result, portArgs *[]string, verboseCount *int) ([]*cobra.Command, error) {
	commands := make([]*cobra.Command, 0)

	for i, instance := range instances {
		e := env
		inst := instance
		index := strconv.Itoa(i + 1)

		instanceC := &cobra.Command{
			Use:   index,
			Short: fmt.Sprintf("ssh to %s %q (%s)", grp, inst.Name, inst.IPAddress),
			RunE: func(cmd *cobra.Command, args []string) error {
				return ssh.Launch(cfg, e, inst, portArgs, verboseCount, args)
			},
		}

		commands = append(commands, instanceC)
	}
	return commands, nil
}

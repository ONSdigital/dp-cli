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


// Example structure of the ssh command:
//
// 	ssh command
// 		environment command
// 			group command
//				instance index
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

func createEnvironmentGroupCommands(env config.Environment, cfg *config.Config) ([]*cobra.Command, error) {
	groups, err := ansible.GetGroupsForEnvironment(cfg.DPSetupPath, env.Name)
	if err != nil {
		return nil, errors.WithMessagef(err, "error loading ansible hosts for %s\n", env.Name)
	}

	profile := ""
	commands := make([]*cobra.Command, 0)

	for _, grp := range groups {

		instances, err := aws.ListEC2ByAnsibleGroup(env.Name, profile, grp)
		if err != nil {
			return nil, errors.WithMessagef(err, "error fetching ec2: %q", env)
		}

		instCount := len(instances)
		commands = append(commands, &cobra.Command{
			Use:   grp,
			Short: fmt.Sprintf("ssh to a %s instance in %s - %d available", grp, env.Name, instCount),
			Args:  validateIndexChoice(env.Name, profile, grp),
			RunE:  newEnvRunFunc(cfg.SSHConfig.User, profile, grp, env, instances),
		})
	}

	return commands, nil
}


func newEnvRunFunc(sshUser, profile, grp string, env config.Environment, instances []aws.EC2Result) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		log := getLogger(env)

		if len(sshUser) == 0 {
			return errors.New("DP_SSH_USER environment variable must be set")
		}

		log("ssh to %s", env.Name)

		choice, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("specify an integer value for instance index in range 1..%d", choice)
		}

		for _, v := range instances {
			out.InfoFHighlight("ListEC2ByAnsibleGroup %s\n", v.Name)
		}
		return nil
	}
}

func validateIndexChoice(env, profile, grp string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		r, err := aws.ListEC2ByAnsibleGroup(env, profile, grp)
		if err != nil {
			return errors.WithMessagef(err, "error fetching ec2: %q", env)
		}

		available := len(r)

		if available == 0 {
			return fmt.Errorf("no matching ec2 instances found for env: %q, profile: %q, group: %q", env, profile, grp)
		}

		if len(args) == 0 {
			return fmt.Errorf("specify and instance index in range 1..%d", available)
		}

		choice, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("specify an integer value for instance index in range 1..%d", available)
		}

		if choice <= 0 || choice > available {
			return errors.Errorf("specify an integer value for instance index in range 1..%d", available)
		}
		return nil
	}
}

func getLogger(env config.Environment) out.Log {
	logFunc := out.InfoFHighlight
	if env.Name == "production" {
		logFunc = out.ErrorFHighlight
	}
	return logFunc
}

package command

import (
	"fmt"
	"strconv"

	"github.com/ONSdigital/dp-cli/ansible"
	"github.com/ONSdigital/dp-cli/aws"
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"
	"github.com/ONSdigital/dp-cli/scp"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// scpCommand builds a cobra.Command to SCP into an environment
//
// The command-line has the following structure:
//
// 	scp		# dp
// 	 environment 	# develop
// 	  group		# publishing_mount
//	   instance	# 1
//	    [--pull]
//	     <fromFile>
//	      <toFile>
//
func scpCommand(cfg *config.Config) (*cobra.Command, error) {
	scpC := &cobra.Command{
		Use:   "scp",
		Short: "Push (or `--pull`) a file to (from) an environment using scp",
	}

	pullFlag := scpC.PersistentFlags().Bool("pull", false, "pull file - first arg is remote-file [default: push (1st arg local)]")
	verboseCount := scpC.PersistentFlags().CountP("verbose", "v", "verbose - increase scp verbosity")
	recurse := scpC.PersistentFlags().BoolP("recurse", "r", false, "recurse - copy recursively")
	environmentCommands, err := createEnvironmentSCPSubCommands(cfg, pullFlag, recurse, verboseCount)
	if err != nil {
		return nil, err
	}
	if len(environmentCommands) == 0 {
		out.Warn("Warning: No sub-commands found for envs - missing envs in config?")
	}

	scpC.AddCommand(environmentCommands...)
	return scpC, nil
}

// create an array of environment sub-commands available to `scp`
func createEnvironmentSCPSubCommands(cfg *config.Config, isPull, recurse *bool, verboseCount *int) ([]*cobra.Command, error) {
	commands := make([]*cobra.Command, 0)

	for _, env := range cfg.Environments {
		envC := &cobra.Command{
			Use:   env.Name,
			Short: "scp on " + env.Name,
		}

		groupCommands, err := createEnvironmentGroupSCPSubCommands(env, cfg, isPull, recurse, verboseCount)
		if err != nil {
			return nil, errors.WithMessagef(err, "error creating group commands for env: %s", env.Name)
		}

		envC.AddCommand(groupCommands...)
		commands = append(commands, envC)
	}
	return commands, nil
}

// create an array of environment group sub-commands available to `scp env`
func createEnvironmentGroupSCPSubCommands(env config.Environment, cfg *config.Config, isPull, recurse *bool, verboseCount *int) ([]*cobra.Command, error) {
	groups, err := ansible.GetGroupsForEnvironment(cfg.DPSetupPath, env.Name)
	if err != nil {
		return nil, errors.WithMessagef(err, "error loading ansible hosts for %s", env.Name)
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
			Short: fmt.Sprintf("scp on %s %s", env.Name, grp),
		}

		instanceCommands, err := createInstanceSCPSubCommands(grp, cfg, env, instances, isPull, recurse, verboseCount)
		if err != nil {
			return nil, err
		}

		grpC.AddCommand(instanceCommands...)
		commands = append(commands, grpC)
	}

	return commands, nil
}

// create an array of instance sub-commands available to `scp env group`
func createInstanceSCPSubCommands(grp string, cfg *config.Config, env config.Environment, instances []aws.EC2Result, isPull, recurse *bool, verboseCount *int) ([]*cobra.Command, error) {
	commands := make([]*cobra.Command, 0)

	for i, instance := range instances {
		e := env
		inst := instance
		index := strconv.Itoa(i + 1)

		instanceC := &cobra.Command{
			Use:   index + " <srcFile> <destFile>",
			Short: fmt.Sprintf("scp on %q %q (%s)", grp, inst.Name, inst.IPAddress),
			Long: fmt.Sprintf("scp on %q %q (%s) args: <srcFile> <destFile>\n"+
				"By default, <srcFile> is local and pushed to <remoteHost>:<destFile>, "+
				"(but if `scp --pull` was used, <remoteHost>:<srcFile> is pulled).\n"+
				"The remote file can be a relative path (rel. to your remote home dir).",
				grp, inst.Name, inst.IPAddress,
			),
			Args: cobra.ExactValidArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return scp.Launch(cfg, e, inst, isPull, recurse, verboseCount, args[0], args[1])
			},
		}

		commands = append(commands, instanceC)
	}
	return commands, nil
}

package ssh

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ONSdigital/dp-cli/aws"
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"
)

// Launch an ssh connection to the specified environment
func Launch(cfg *config.Config, env config.Environment, instance aws.EC2Result, portArgs *[]string, verboseCount *int, extraArgs []string) error {
	if cfg.SSHUser == nil || len(*cfg.SSHUser) == 0 {
		out.Highlight(out.WARN, "no %s is defined in configuration file (or `--user`) you can view the app configuration values using the %s command", "ssh-user", "spew config")
		return errors.New("missing `ssh-user` in config file (or no `--user`)")
	}

	lvl := out.GetLevel(env)
	out.Highlight(lvl, "Launching SSH connection to %s", env.Name)
	out.Highlight(lvl, "[IP: %s | Name: %s | Id: %s | Groups: %s | AKA: %s", instance.IPAddress, instance.Name, instance.InstanceId, instance.AnsibleGroups, strings.Join(instance.GroupAKA, ", "))

	ansibleDir := cfg.GetAnsibleDirectory(env)

	var userHost string
	args := []string{"-F", "ssh.cfg"}
	sshUser := *cfg.SSHUser
	if len(env.SSHUser) > 0 {
		sshUser = env.SSHUser
	}

	if portArgs != nil {
		for _, portArg := range *portArgs {
			sshPortArgs, err := getSSHPortArguments(portArg)
			if err != nil {
				return err
			}
			args = append(args, sshPortArgs...)
		}
	}
	if env.IsAWSA() {
		userHost = fmt.Sprintf("%s@%s", sshUser, instance.IPAddress)
	} else {
		os.Setenv("AWS_PROFILE", cfg.GetProfile(env.Name))
		userHost = fmt.Sprintf("%s@%s", sshUser, instance.InstanceId)
	}
	for v := 0; v < *verboseCount; v++ {
		args = append(args, "-v")
	}
	args = append(args, userHost)
	args = append(args, extraArgs...)
	fmt.Println(args)
	return execCommand(ansibleDir, "ssh", args...)
}

func execCommand(wrkDir, command string, arg ...string) error {
	c := exec.Command(command, arg...)
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Env = os.Environ()
	c.Dir = wrkDir
	if err := c.Run(); err != nil {
		return err
	}
	return nil
}

func getSSHPortArguments(portArg string) ([]string, error) {
	validPort := regexp.MustCompile(
		`^(?P<local_port>[0-9]+)` +
			`(?:` +
			`(?:` + `:(?P<remote_host>[-a-z0-9._]+)` + `)?` +
			`(?:` + `:(?P<remote_port>[0-9]+)` + `)` +
			`)?$`,
	)
	if !validPort.MatchString(portArg) {
		return nil, fmt.Errorf("%q is not a valid port forwarding argument", portArg)
	}

	ports := strings.Split(portArg, ":")
	localPort, host, remotePort := ports[0], "localhost", ports[0]
	if len(ports) == 2 {
		remotePort = ports[1]
	} else if len(ports) == 3 {
		host, remotePort = ports[1], ports[2]
	}
	sshPortArg := fmt.Sprintf("%s:%s:%s", localPort, host, remotePort)
	return []string{"-L", sshPortArg}, nil
}

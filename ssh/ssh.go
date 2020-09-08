package ssh

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ONSdigital/dp-cli/aws"
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"
)

// Launch an ssh connection to the specified environment
func Launch(cfg *config.Config, env config.Environment, instance aws.EC2Result, portArgs *[]string, verboseCount *int, extraArgs []string) error {
	if len(cfg.SSHUser) == 0 {
		out.Highlight(out.WARN, "no %s is defined in your configuration file you can view the app configuration values using the %s command", "ssh user", "spew config")
		return errors.New("missing `ssh user` in config file")
	}

	lvl := out.GetLevel(env)
	fmt.Println("")
	out.Highlight(lvl, "Launching SSH connection to %s", env.Name)
	out.Highlight(lvl, "[IP: %s | Name: %s | Groups %s]", instance.IPAddress, instance.Name, instance.AnsibleGroups)

	ansibleDir := filepath.Join(cfg.SourceDir, "dp-setup", "ansible")
	args := []string{"-F", "ssh.cfg"}
	if portArgs != nil {
		for _, portArg := range *portArgs {
			sshPortArgs, err := getSSHPortArguments(portArg)
			if err != nil {
				return err
			}
			args = append(args, sshPortArgs...)
		}
	}
	for v := 0; v < *verboseCount; v++ {
		args = append(args, "-v")
	}
	userHost := fmt.Sprintf("%s@%s", cfg.SSHUser, instance.IPAddress)
	args = append(args, userHost)
	args = append(args, extraArgs...)
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

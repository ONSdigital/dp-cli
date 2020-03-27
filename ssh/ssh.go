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

// Open an ssh connect to the specified environment
func Launch(cfg *config.Config, env config.Environment, instance aws.EC2Result, portArgs *[]string) error {
	if len(cfg.SSHUser) == 0 {
		out.Highlight(out.WARN, "no %s is defined in your configuration file you can view the app configuration values using the %s command\n", "ssh user", "spew config")
		return errors.New("missing ssh user in config file")
	}

	lvl := out.GetLevel(env)
	fmt.Println("")
	out.Highlight(lvl, "Launching SSH connection to %s", env.Name)
	out.Highlight(lvl, "[IP: %s | Name: %s | Groups %s]\n", instance.IPAddress, instance.Name, instance.AnsibleGroups)

	pwd := filepath.Join(cfg.DPSetupPath, "ansible")
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
	unixUser := fmt.Sprintf("%s@%s", cfg.SSHUser, instance.IPAddress)
	args = append(args, unixUser)
	return execCommand(pwd, "ssh", args...)
}

func execCommand(pwd string, command string, arg ...string) error {
	c := exec.Command(command, arg...)
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Env = os.Environ()
	c.Dir = pwd
	if err := c.Run(); err != nil {
		return err
	}
	return nil
}

func getSSHPortArguments(portArg string) ([]string, error) {
	validPort := regexp.MustCompile(`^(([0-9]+):)?([0-9]+)$`)
	if !validPort.MatchString(portArg) {
		return nil, errors.New(fmt.Sprintf("'%s' is not a valid port forwarding argument", portArg))
	}

	ports := strings.Split(portArg, ":")
	var sshPortArg string
	if len(ports) == 1 {
		sshPortArg = fmt.Sprintf("%s:localhost:%s", ports[0], ports[0])
	} else {
		sshPortArg = fmt.Sprintf("%s:localhost:%s", ports[0], ports[1])
	}

	return []string{"-L", sshPortArg}, nil
}

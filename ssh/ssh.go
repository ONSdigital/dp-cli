package ssh

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ONSdigital/dp-cli/aws"
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"
)

// Open an ssh connect to the specified environment
func Launch(cfg *config.Config, env config.Environment, instance aws.EC2Result) error {
	if len(cfg.SSHUser) == 0 {
		out.Highlight(out.WARN, "no %s is defined in your configuration file you can view the app configuration values using the %s command\n", "ssh user", "spew config")
		return errors.New("missing ssh user in config file")
	}

	lvl := out.GetLevel(env)
	fmt.Println("")
	out.Highlight(lvl, "Launching SSH connection to %s", env.Name)
	out.Highlight(lvl, "[IP: %s | Name: %s | Groups %s]\n", instance.IPAddress, instance.Name, instance.AnsibleGroups)

	pwd := filepath.Join(cfg.DPSetupPath, "ansible")
	unixUser := fmt.Sprintf("%s@%s", cfg.SSHUser, instance.IPAddress)
	return execCommand(pwd, "ssh", "-F", "ssh.cfg", unixUser)
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

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
	if len(cfg.SSHConfig.User) == 0 {
		out.WarnFHighlight("no %s is defined in your configuration file you can view the app configuration values using the %s command\n", "ssh user", "spew config")
		return errors.New("missing ssh user in config file")
	}

	logFunc := getLogger(env)
	fmt.Println("")
	logFunc("Launching SSH connection to  %s", env.Name)
	logFunc("[IP: %s | Name: %s | Groups %s]\n", instance.IPAddress, instance.Name, instance.AnsibleGroups)

	pwd := filepath.Join(cfg.DPSetupPath, "ansible")
	unixUser := fmt.Sprintf("%s@%s", cfg.SSHConfig.User, instance.IPAddress)
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

func getLogger(env config.Environment) out.Log {
	logFunc := out.InfoFHighlight
	if env.Name == "production" {
		logFunc = out.ErrorFHighlight
	}
	return logFunc
}

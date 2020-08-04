package scp

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

func withCWD(file string) (string, error) {
	if file[0] == '/' {
		return file, nil
	}
	var pwd string
	var err error
	if pwd, err = os.Getwd(); err != nil {
		return "", err
	}
	return filepath.Join(pwd, file), nil
}

// Launch an scp file copy to the specified environment
func Launch(cfg *config.Config, env config.Environment, instance aws.EC2Result, isPull, recurse *bool, verboseCount *int, srcFile, destFile string) (err error) {
	if len(cfg.SSHUser) == 0 {
		out.Highlight(out.WARN, "no %s is defined in your configuration file you can view the app configuration values using the %s command", "ssh user", "spew config")
		return errors.New("missing `ssh user` in config file")
	}

	verb := "pushing"
	if *isPull {
		verb = "pulling"
		srcFile = fmt.Sprintf("%s@%s:%s", cfg.SSHUser, instance.IPAddress, srcFile)
		if destFile, err = withCWD(destFile); err != nil {
			out.Highlight(out.WARN, "could not determine your cwd")
			return err
		}
	} else {
		destFile = fmt.Sprintf("%s@%s:%s", cfg.SSHUser, instance.IPAddress, destFile)
		if srcFile, err = withCWD(srcFile); err != nil {
			out.Highlight(out.WARN, "could not determine your cwd")
			return err
		}

	}

	lvl := out.GetLevel(env)
	out.Highlight(lvl, "SCP %s for %s (%s -> %s)", verb, env.Name, srcFile, destFile)
	out.Highlight(lvl, "[IP: %s | Name: %s | Groups %s]", instance.IPAddress, instance.Name, instance.AnsibleGroups)

	ansibleDir := filepath.Join(cfg.DPSetupPath, "ansible")
	flags := "-p"
	for v := 0; v < *verboseCount; v++ {
		flags += "v"
	}
	if *recurse {
		flags += "r"
	}
	return execCommand(ansibleDir, "scp", flags+"F", "ssh.cfg", srcFile, destFile)
}

func execCommand(pwd, command string, arg ...string) error {
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

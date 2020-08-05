package scp

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ONSdigital/dp-cli/aws"
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"
)

// Options holds the state of flags given
type Options struct {
	IsPull      *bool
	IsRecursing *bool
	IsConfirmed *bool
	Verbosity   *int
}

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

// Launch an scp file copy to/from the specified environment
func Launch(cfg *config.Config, env config.Environment, instance aws.EC2Result, opts Options, srcFile, destFile string) (err error) {
	if len(cfg.SSHUser) == 0 {
		out.Highlight(out.WARN, "no %s is defined in your configuration file you can view the app configuration values using the %s command", "ssh user", "spew config")
		return errors.New("missing `ssh user` in config file")
	}

	verb := "pushing"
	if *opts.IsPull {
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
		if _, err := os.Stat(srcFile); err != nil {
			out.Highlight(out.WARN, "could not access source file: %s", srcFile)
			return err
		}
	}

	lvl := out.GetLevel(env)
	out.Highlight(lvl, "SCP %s for %s (%s -> %s)", verb, env.Name, srcFile, destFile)
	out.Highlight(lvl, "[IP: %s | Name: %s | Groups %s]", instance.IPAddress, instance.Name, instance.AnsibleGroups)

	if *opts.IsPull && env.Name == "production" && !*opts.IsConfirmed {
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("Legal declaration: I confirm that I am NOT copying sensitive files (yes/no): ")
			yorn, _ := reader.ReadString('\n')
			if yorn == "yes\n" {
				break
			} else if yorn == "no\n" {
				return errors.New("failed to confirm legal declaration")
			}
		}
	}

	ansibleDir := filepath.Join(cfg.DPSetupPath, "ansible")
	flags := "-p"
	for v := 0; v < *opts.Verbosity; v++ {
		flags += "v"
	}
	if *opts.IsRecursing {
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

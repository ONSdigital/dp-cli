package scp

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	var currentDir string
	var err error
	if currentDir, err = os.Getwd(); err != nil {
		return "", err
	}
	return filepath.Join(currentDir, file), nil
}

// Launch an scp file copy to/from the specified environment
func Launch(cfg *config.Config, env config.Environment, instance aws.EC2Result, opts Options, srcFiles []string, target string) (err error) {
	if cfg.SSHUser == nil || len(*cfg.SSHUser) == 0 {
		out.Highlight(out.WARN, "no %s is defined in your configuration file you can view the app configuration values using the %s command", "ssh-user", "spew config")
		return errors.New("missing `ssh-user` in config file")
	}

	ansibleDir := cfg.GetAnsibleDirectory(env)

	flags := "-p"
	for v := 0; v < *opts.Verbosity; v++ {
		flags += "v"
	}
	if *opts.IsRecursing {
		flags += "r"
	}
	cmdArgs := []string{flags + "F", "ssh.cfg"}
	if env.IsCI() {
		cmdArgs = []string{}
	}
	sshUser := *cfg.SSHUser
	if len(env.SSHUser) > 0 {
		sshUser = env.SSHUser
	}
	for _, srcFile := range srcFiles {
		if *opts.IsPull {
			if env.IsAWSA() {
				srcFile = fmt.Sprintf("%s@%s:%s", sshUser, instance.IPAddress, srcFile)
			} else {
				os.Setenv("AWS_PROFILE", env.Profile)
				srcFile = fmt.Sprintf("%s@%s:%s", sshUser, instance.InstanceId, srcFile)
			}
		} else {
			if srcFile, err = withCWD(srcFile); err != nil {
				out.Highlight(out.WARN, "could not determine your cwd")
				return err
			}
			if _, err := os.Stat(srcFile); err != nil {
				out.Highlight(out.WARN, "could not access source file: %s", srcFile)
				return err
			}
		}
		cmdArgs = append(cmdArgs, srcFile)
	}
	verb := "pushing"
	if *opts.IsPull {
		verb = "pulling"
		if target, err = withCWD(target); err != nil {
			out.Highlight(out.WARN, "could not determine your cwd")
			return err
		}
	} else {
		if env.IsAWSA() {
			target = fmt.Sprintf("%s@%s:%s", sshUser, instance.IPAddress, target)
		} else {
			os.Setenv("AWS_PROFILE", env.Profile)
			target = fmt.Sprintf("%s@%s:%s", sshUser, instance.InstanceId, target)
		}
	}
	cmdArgs = append(cmdArgs, target)

	lvl := out.GetLevel(env)
	out.Highlight(lvl, "SCP %s for %s (%s -> %s)", verb, env.Name, strings.Join(srcFiles, ", "), target)
	out.Highlight(lvl, "[IP: %s | Name: %s | Id %s | Groups %s | AKA %s]", instance.IPAddress, instance.Name, instance.InstanceId, instance.AnsibleGroups, strings.Join(instance.GroupAKA, ", "))

	if *opts.IsPull && env.IsLive() && !*opts.IsConfirmed {
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

	return execCommand(ansibleDir, "scp", cmdArgs...)
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

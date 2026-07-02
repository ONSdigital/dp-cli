package cli

import (
	"context"
	"os"
	"os/exec"
	"time"

	"github.com/ONSdigital/dp-cli/out"
)

func ExecCommand(ctx context.Context, command, wrkDir string) error {
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if wrkDir != "" {
		cmd.Dir = wrkDir
	}

	return cmd.Run()
}

func GetProgressTicker() (stopChan chan bool, cleanup func()) {
	stopC := make(chan bool)

	progressTicker := func() {
		done := false

		for !done {
			select {
			case <-stopC:
				done = true
			default:
				out.InfoAppend(".")
				time.Sleep(time.Second * 1)
			}
		}
	}

	return stopC, progressTicker
}

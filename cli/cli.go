package cli

import (
	"dp-cli/out"
	"os"
	"os/exec"
	"time"
)

func ExecCommand(command string, wrkDir string) error {
	cmd := exec.Command("bash", "-c", command)
	cmd.Stderr = os.Stderr

	if len(wrkDir) > 0 {
		cmd.Dir = wrkDir
	}

	return cmd.Run()
}

func GetProgressTicker() (chan bool, func()) {
	stopC := make(chan bool, 0)

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

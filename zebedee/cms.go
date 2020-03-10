package zebedee

import (
	"bufio"
	"os"

	"github.com/ONSdigital/dp-cli/out"
)

func NewCMS() error {
	if len(zebedeeRoot) == 0 {
		out.WriteF(out.INFO, "env var %s not defined\n", "zebedee_root")
	} else {
		out.Highlight(out.INFO, "zebedee_root=%s\n", zebedeeRoot)
	}

	getContentDir()
	return nil
}

func getContentDir() string {
	var dir string
	s := bufio.NewScanner(os.Stdin)
	out.Write(out.INFO, "Enter the directory path where the CMS folder structure should be created:")
	out.Prompt()
	for s.Scan() {
		dir = s.Text()
		if len(dir) == 0 {
			out.Write(out.WARN, "Enter the directory path where the CMS folder structure should be created:")
			out.Prompt()
			continue
		}
		break
	}

	out.Highlight(out.INFO, "creating CMS folder structure under %s", dir)
	return dir
}

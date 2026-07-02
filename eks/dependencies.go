package eks

import (
	"os/exec"
)

// Dependency represents a required external tool
type Dependency struct {
	Name        string
	Command     string
	InstallHint string
}

// SessionDependencies lists all external tools needed for EKS tunnel operations
var SessionDependencies = []Dependency{
	{
		Name:        "AWS CLI",
		Command:     "aws",
		InstallHint: "Install from https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html",
	},
	{
		Name:        "SSM Session Manager Plugin",
		Command:     "session-manager-plugin",
		InstallHint: "Install from https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html",
	},
	{
		Name:        "socat",
		Command:     "socat",
		InstallHint: "Install with: brew install socat",
	},
}

// TODO: Currently this is for EKS but can be elevated for use across all sub cmds
// CheckDependencies verifies all required tools are available for the sub cmd.
// Returns a list of missing dependencies.
func CheckDependencies() []Dependency {
	var missing []Dependency
	for _, dep := range SessionDependencies {
		if _, err := exec.LookPath(dep.Command); err != nil {
			missing = append(missing, dep)
		}
	}
	return missing
}

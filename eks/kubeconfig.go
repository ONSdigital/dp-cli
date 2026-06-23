package eks

import (
	"os/exec"
	"strings"
)

// UpdateKubeconfig runs aws eks update-kubeconfig for the given cluster.
// The context alias is always the cluster name (one context per cluster).
// The user-alias includes the roleSuffix to ensure each role gets its own
// credential entry, preventing profile cross-contamination.
func UpdateKubeconfig(clusterName, profile, roleSuffix string) (string, error) {
	userAlias := clusterName + roleSuffix
	args := []string{
		"eks", "update-kubeconfig",
		"--name", clusterName,
		"--alias", clusterName,
		"--user-alias", userAlias,
	}
	if profile != "" {
		args = append(args, "--profile", profile)
	}

	cmd := exec.Command("aws", args...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

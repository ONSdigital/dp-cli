package eks

import (
	"os/exec"
	"strings"
)

// UpdateKubeconfig runs aws eks update-kubeconfig for the given cluster.
// Region is determined by the AWS profile configuration.
func UpdateKubeconfig(clusterName, profile string) (string, error) {
	args := []string{
		"eks", "update-kubeconfig",
		"--name", clusterName,
		"--alias", clusterName,
	}
	if profile != "" {
		args = append(args, "--profile", profile)
	}

	cmd := exec.Command("aws", args...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

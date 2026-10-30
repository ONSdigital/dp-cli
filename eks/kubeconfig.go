package eks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"
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

	cmd := exec.CommandContext(context.Background(), "aws", args...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// KubeconfigPath returns the path to the active kubeconfig file. It honours the
// first entry of the KUBECONFIG environment variable (colon-separated, as kubectl
// treats it) and falls back to ~/.kube/config.
func KubeconfigPath() (string, error) {
	if env := os.Getenv("KUBECONFIG"); env != "" {
		// KUBECONFIG may contain multiple colon-separated paths; aws
		// update-kubeconfig writes to the first one.
		first := strings.Split(env, string(os.PathListSeparator))[0]
		if first != "" {
			return first, nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}
	return filepath.Join(home, ".kube", "config"), nil
}

// PatchKubeconfigForNoSudo rewrites the named cluster entry so kubectl connects
// directly to the local SSM port-forward instead of relying on socat/hosts to
// re-present the tunnel on port 443.
//
// It sets:
//   - server:          https://127.0.0.1:<localPort>
//   - tls-server-name: <endpoint>  (the real EKS API hostname)
//
// The existing certificate-authority-data written by `aws eks update-kubeconfig`
// is preserved. tls-server-name makes kubectl send the real hostname as the TLS
// SNI and validate the server certificate against it, so TLS remains fully
// verified with no insecure-skip-tls-verify.
func PatchKubeconfigForNoSudo(clusterName, endpoint string, localPort int) error {
	path, err := KubeconfigPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path) //nolint:gosec // G304 - path is the user's kubeconfig, resolved above
	if err != nil {
		return fmt.Errorf("failed to read kubeconfig %s: %w", path, err)
	}

	var doc yaml.MapSlice
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("failed to parse kubeconfig %s: %w", path, err)
	}

	patched, err := patchClusterEntry(doc, clusterName, endpoint, localPort)
	if err != nil {
		return err
	}

	out, err := yaml.Marshal(patched)
	if err != nil {
		return fmt.Errorf("failed to serialise kubeconfig: %w", err)
	}

	if err := os.WriteFile(path, out, 0600); err != nil {
		return fmt.Errorf("failed to write kubeconfig %s: %w", path, err)
	}
	return nil
}

// patchClusterEntry walks the kubeconfig document (parsed as an ordered
// yaml.MapSlice to preserve key ordering and untouched fields) and updates the
// cluster block for clusterName.
//
// aws eks update-kubeconfig names the *context* after --alias (the cluster
// name), but the *cluster* entry is named after the cluster ARN. So we first
// resolve the context named clusterName to find the real cluster-entry name
// (contexts[].context.cluster), then patch the matching clusters[] item. If no
// such context exists we fall back to matching the clusters[] item by name
// directly (covers custom/legacy kubeconfigs).
func patchClusterEntry(doc yaml.MapSlice, clusterName, endpoint string, localPort int) (yaml.MapSlice, error) {
	server := fmt.Sprintf("https://127.0.0.1:%d", localPort)

	// Resolve the cluster-entry name via the context, defaulting to clusterName.
	targetClusterRef := resolveClusterRefFromContext(doc, clusterName)

	for i, top := range doc {
		key, ok := top.Key.(string)
		if !ok || key != "clusters" {
			continue
		}
		clusters, ok := top.Value.([]interface{})
		if !ok {
			return nil, fmt.Errorf("kubeconfig clusters entry is not a list")
		}

		found := false
		for _, c := range clusters {
			entry, ok := c.(yaml.MapSlice)
			if !ok {
				continue
			}
			name := getMapSliceString(entry, "name")
			if name != targetClusterRef && name != clusterName {
				continue
			}
			cluster, idx := getMapSliceValue(entry, "cluster")
			clusterMap, ok := cluster.(yaml.MapSlice)
			if !ok || idx < 0 {
				return nil, fmt.Errorf("cluster %q has no cluster block", name)
			}
			clusterMap = setMapSliceString(clusterMap, "server", server)
			clusterMap = setMapSliceString(clusterMap, "tls-server-name", endpoint)
			entry[idx].Value = clusterMap
			found = true
		}
		if !found {
			return nil, fmt.Errorf("cluster %q not found in kubeconfig (context ref %q)", clusterName, targetClusterRef)
		}
		doc[i].Value = clusters
		return doc, nil
	}
	return nil, fmt.Errorf("no clusters section found in kubeconfig")
}

// resolveClusterRefFromContext returns the cluster reference (the value of
// contexts[name==ctxName].context.cluster) for the given context name. If the
// context or field cannot be found it returns ctxName unchanged.
func resolveClusterRefFromContext(doc yaml.MapSlice, ctxName string) string {
	for _, top := range doc {
		if k, ok := top.Key.(string); !ok || k != "contexts" {
			continue
		}
		contexts, ok := top.Value.([]interface{})
		if !ok {
			return ctxName
		}
		for _, c := range contexts {
			entry, ok := c.(yaml.MapSlice)
			if !ok || getMapSliceString(entry, "name") != ctxName {
				continue
			}
			ctxVal, _ := getMapSliceValue(entry, "context")
			if ctxMap, ok := ctxVal.(yaml.MapSlice); ok {
				if ref := getMapSliceString(ctxMap, "cluster"); ref != "" {
					return ref
				}
			}
		}
	}
	return ctxName
}

// getMapSliceString returns the string value for key in a yaml.MapSlice, or "".
func getMapSliceString(m yaml.MapSlice, key string) string {
	for _, item := range m {
		if k, ok := item.Key.(string); ok && k == key {
			if s, ok := item.Value.(string); ok {
				return s
			}
		}
	}
	return ""
}

// getMapSliceValue returns the value for key and its index, or (nil, -1).
func getMapSliceValue(m yaml.MapSlice, key string) (value interface{}, index int) {
	for i, item := range m {
		if k, ok := item.Key.(string); ok && k == key {
			return item.Value, i
		}
	}
	return nil, -1
}

// setMapSliceString sets key=value, updating in place if present or appending.
func setMapSliceString(m yaml.MapSlice, key, value string) yaml.MapSlice {
	for i, item := range m {
		if k, ok := item.Key.(string); ok && k == key {
			m[i].Value = value
			return m
		}
	}
	return append(m, yaml.MapItem{Key: key, Value: value})
}

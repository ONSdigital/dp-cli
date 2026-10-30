package eks

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	hostsMarker = "# EKS-TUNNEL-MANAGED"

	// DefaultStateDir is where per-cluster tunnel state files are stored unless
	// overridden via config. It is under /tmp so it is ephemeral and cleared on
	// reboot, which suits transient tunnel state.
	DefaultStateDir = "/tmp/eks-tunnels-ssm"
	// DefaultBasePort and DefaultMaxPort bound the local port range used for SSM
	// port-forwards (inclusive). The range must not overlap privileged ports.
	DefaultBasePort = 9443
	DefaultMaxPort  = 9500
)

// These are vars (not consts) so they can be overridden via Configure (from
// user config) and pointed at temp values in tests.
var (
	tunnelDir = DefaultStateDir
	basePort  = DefaultBasePort
	maxPort   = DefaultMaxPort
)

// Configure applies optional overrides for the tunnel state directory and local
// port range. Empty/zero values fall back to the defaults, so callers can pass
// raw config values without pre-checking them. An invalid range (base > max) is
// ignored in favour of the defaults.
func Configure(stateDir string, base, maxP int) {
	if stateDir != "" {
		tunnelDir = stateDir
	}
	if base > 0 {
		basePort = base
	}
	if maxP > 0 {
		maxPort = maxP
	}
	// Guard against a misconfigured inverted range.
	if basePort > maxPort {
		basePort = DefaultBasePort
		maxPort = DefaultMaxPort
	}
}

// TunnelState represents the persisted state of an active tunnel. It is stored
// as a single JSON file per cluster under tunnelDir (<cluster>.json).
type TunnelState struct {
	ClusterName string `json:"clusterName"`
	SSMPid      int    `json:"ssmPid"`
	SocatPid    int    `json:"socatPid"`
	Endpoint    string `json:"endpoint"`
	LoopbackIP  string `json:"loopbackIP"`
	IPv4        string `json:"ipv4"`
	LocalPort   int    `json:"localPort"`
	// NoSudo indicates the tunnel was created in no-sudo mode: no socat, no
	// /etc/hosts entry, no loopback alias. kubectl connects directly to
	// 127.0.0.1:LocalPort via a patched kubeconfig (tls-server-name).
	NoSudo bool `json:"noSudo"`
}

// EnsureTunnelDir creates the tunnel state directory if it doesn't exist
func EnsureTunnelDir() error {
	return os.MkdirAll(tunnelDir, 0755)
}

// AllocateLocalPort finds an available port in the range basePort-maxPort (inclusive).
//
// It probes with net.Listen (no external tools, portable) and closes the
// listener before returning, so callers that then hand the port to a child
// process (the SSM session) can bind it. This leaves a small TOCTOU window
// between allocation and the child binding; for concurrent multi-cluster starts
// use AllocateAndHoldLocalPort instead, which keeps the port reserved until the
// caller is ready to hand off.
func AllocateLocalPort() (int, error) {
	port, ln, err := AllocateAndHoldLocalPort()
	if err != nil {
		return 0, err
	}
	_ = ln.Close() //nolint:errcheck // releasing our probe listener
	return port, nil
}

// AllocateAndHoldLocalPort finds an available port and returns it together with
// an open listener holding that port. The caller MUST close the returned
// listener immediately before starting the process that will bind the port.
//
// Holding the listener keeps the port reserved against other allocations in the
// same run (the OS will not hand the same port to another net.Listen while it is
// open), which prevents two clusters being handed the same port during a
// concurrent/sequential multi-cluster start.
func AllocateAndHoldLocalPort() (int, net.Listener, error) {
	var lc net.ListenConfig
	for port := basePort; port <= maxPort; port++ {
		ln, err := lc.Listen(context.Background(), "tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			// Port is in use (or otherwise unbindable) — try the next one.
			continue
		}
		return port, ln, nil
	}
	return 0, nil, fmt.Errorf("no available ports in range %d-%d", basePort, maxPort)
}

// AllocateLoopbackIP finds the next available loopback IP where port 443 is not in use.
// Checks 127.0.0.1 through 127.0.0.254.
// Uses sudo lsof to detect root-owned processes (socat runs as root).
func AllocateLoopbackIP() (string, error) {
	for i := 1; i <= 254; i++ {
		ip := fmt.Sprintf("127.0.0.%d", i)
		// Check if port 443 is already bound on this IP (need sudo to see root processes)
		cmd := exec.CommandContext(context.Background(), "sudo", "lsof", "-i", fmt.Sprintf("@%s:443", ip)) //nolint:gosec // G204 - ip is internally resolved
		if err := cmd.Run(); err != nil {
			// lsof returns non-zero if nothing is bound — this IP is available
			return ip, nil
		}
	}
	return "", fmt.Errorf("no available loopback IP found (all 127.0.0.1-254 have port 443 in use)")
}

// EnsureLoopbackAlias creates a loopback alias for addresses other than 127.0.0.1
func EnsureLoopbackAlias(ip string) error {
	if ip == "127.0.0.1" {
		return nil
	}

	switch runtime.GOOS {
	case "darwin":
		out, err := exec.CommandContext(context.Background(), "ifconfig", "lo0").Output()
		if err != nil {
			return fmt.Errorf("failed to check loopback interfaces: %w", err)
		}
		if strings.Contains(string(out), "inet "+ip+" ") {
			return nil
		}
		cmd := exec.CommandContext(context.Background(), "sudo", "ifconfig", "lo0", "alias", ip)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case "linux":
		out, err := exec.CommandContext(context.Background(), "ip", "addr", "show", "dev", "lo").Output()
		if err != nil {
			return fmt.Errorf("failed to check loopback interfaces: %w", err)
		}
		if strings.Contains(string(out), " "+ip+"/") {
			return nil
		}
		cmd := exec.CommandContext(context.Background(), "sudo", "ip", "addr", "add", ip+"/8", "dev", "lo") //nolint:gosec // G204 - ip is internally allocated loopback
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// StartSSMPortForward starts an SSM port forwarding session in the background.
func StartSSMPortForward(bastionID, targetIP string, localPort int, profile string) (int, error) {
	logFile := filepath.Join(tunnelDir, fmt.Sprintf("ssm-%d.log", localPort))

	args := []string{
		"ssm", "start-session",
		"--target", bastionID,
		"--document-name", "AWS-StartPortForwardingSessionToRemoteHost",
		//nolint:gocritic // sprintfQuotedString - building JSON, not display strings
		"--parameters", fmt.Sprintf(`{"host":["%s"],"portNumber":["443"],"localPortNumber":["%d"]}`, targetIP, localPort),
	}
	if profile != "" {
		args = append(args, "--profile", profile)
	}

	cmd := exec.CommandContext(context.Background(), "aws", args...)

	f, err := os.Create(logFile)
	if err != nil {
		return 0, fmt.Errorf("failed to create log file: %w", err)
	}
	cmd.Stdout = f
	cmd.Stderr = f

	if err := cmd.Start(); err != nil {
		f.Close()
		return 0, fmt.Errorf("failed to start SSM session: %w", err)
	}
	f.Close()

	// Wait for session to establish
	pid := cmd.Process.Pid
	for i := 0; i < 15; i++ {
		time.Sleep(1 * time.Second)
		content, _ := os.ReadFile(logFile)
		if strings.Contains(string(content), "Waiting for connections") {
			return pid, nil
		}
		// Check if process is still alive
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			content, _ := os.ReadFile(logFile)
			return 0, fmt.Errorf("SSM session died: %s", string(content))
		}
	}

	return pid, nil // Return pid even if we didn't see the message — it might still be starting
}

// StartSocat starts socat to bind a loopback IP on port 443 to a local port
func StartSocat(loopbackIP string, localPort int) (int, error) {
	listenAddr := fmt.Sprintf("TCP-LISTEN:443,bind=%s,fork,reuseaddr", loopbackIP)
	targetAddr := fmt.Sprintf("TCP:127.0.0.1:%d", localPort)

	cmd := exec.CommandContext(context.Background(), "sudo", "socat", listenAddr, targetAddr)
	cmd.Stdout = nil
	cmd.Stderr = nil
	// Detach from the controlling terminal so socat survives after dp exits
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start socat: %w", err)
	}

	// Give socat a moment to bind
	time.Sleep(500 * time.Millisecond)

	// Find the actual socat PID (sudo may have forked it)
	pid := findSocatPid(loopbackIP, localPort)
	if pid > 0 {
		return pid, nil
	}

	// Fallback to the sudo PID if we can't find the socat process
	return cmd.Process.Pid, nil
}

// findSocatPid finds the PID of a socat process bound to the given IP on port 443
// that forwards to the specified local port (to avoid matching unrelated socat instances)
func findSocatPid(loopbackIP string, localPort int) int {
	// Match socat processes with both our bind address AND our target port
	pattern := fmt.Sprintf("socat.*bind=%s.*127.0.0.1:%d", loopbackIP, localPort)
	out, err := exec.CommandContext(context.Background(), "pgrep", "-f", pattern).Output()
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) > 0 && lines[0] != "" {
		pid, err := strconv.Atoi(lines[0])
		if err == nil {
			return pid
		}
	}
	return 0
}

// AddHostsEntry adds an entry to /etc/hosts for the given IP and hostname.
// It guards against a hosts file that does not end in a newline by prepending
// one, so the new entry is always written on its own line.
func AddHostsEntry(ip, hostname, clusterName string) error {
	// Remove any existing entry first
	RemoveHostsEntry(hostname)

	// Prepend a newline if the existing file does not end with one, so we never
	// append onto a partial last line.
	leading := ""
	if content, err := os.ReadFile("/etc/hosts"); err == nil {
		leading = hostsLeadingNewline(content)
	}

	entry := fmt.Sprintf("%s%s %s %s %s\n", leading, ip, hostname, hostsMarker, clusterName)
	cmd := exec.CommandContext(context.Background(), "sudo", "tee", "-a", "/etc/hosts")
	cmd.Stdin = strings.NewReader(entry)
	cmd.Stdout = nil // suppress tee output
	return cmd.Run()
}

// hostsLeadingNewline returns "\n" when content is non-empty and does not already
// end with a newline, so an appended entry starts on its own line.
func hostsLeadingNewline(content []byte) string {
	if n := len(content); n > 0 && content[n-1] != '\n' {
		return "\n"
	}
	return ""
}

// RemoveHostsEntry removes a managed hosts entry for the given hostname
func RemoveHostsEntry(hostname string) {
	content, err := os.ReadFile("/etc/hosts")
	if err != nil {
		return
	}
	var filtered []string
	for _, line := range strings.Split(string(content), "\n") {
		if !strings.Contains(line, hostname) || !strings.Contains(line, hostsMarker) {
			filtered = append(filtered, line)
		}
	}
	cmd := exec.CommandContext(context.Background(), "sudo", "tee", "/etc/hosts")
	cmd.Stdin = strings.NewReader(strings.Join(filtered, "\n"))
	cmd.Stdout = nil
	_ = cmd.Run() //nolint:errcheck // best-effort hosts file update
}

// FlushDNSCache flushes the OS DNS cache. Behaviour is OS-specific and best-effort.
func FlushDNSCache() {
	switch runtime.GOOS {
	case "darwin":
		_ = exec.CommandContext(context.Background(), "sudo", "dscacheutil", "-flushcache").Run()       //nolint:errcheck // best-effort DNS flush
		_ = exec.CommandContext(context.Background(), "sudo", "killall", "-HUP", "mDNSResponder").Run() //nolint:errcheck // best-effort DNS flush
	case "linux":
		// Try resolvectl (systemd 239+), then systemd-resolve, then nscd — all best-effort
		if exec.CommandContext(context.Background(), "resolvectl", "flush-caches").Run() != nil {
			if exec.CommandContext(context.Background(), "systemd-resolve", "--flush-caches").Run() != nil {
				_ = exec.CommandContext(context.Background(), "sudo", "systemctl", "restart", "nscd").Run() //nolint:errcheck // best-effort DNS flush
			}
		}
	}
}

// stateFilePath returns the path to a cluster's single JSON state file.
func stateFilePath(clusterName string) string {
	return filepath.Join(tunnelDir, clusterName+".json")
}

// SaveTunnelState persists tunnel state to a single JSON file per cluster.
// The write is atomic (write to a temp file, then rename) so a crash mid-write
// cannot leave a partially written state file behind.
func SaveTunnelState(state TunnelState) error {
	if err := EnsureTunnelDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tunnel state: %w", err)
	}

	path := stateFilePath(state.ClusterName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("failed to write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp) //nolint:errcheck // best-effort temp cleanup
		return fmt.Errorf("failed to persist %s: %w", path, err)
	}
	return nil
}

// LoadTunnelState reads tunnel state from a cluster's JSON file.
func LoadTunnelState(clusterName string) (*TunnelState, error) {
	data, err := os.ReadFile(stateFilePath(clusterName)) //nolint:gosec // G304 - path built from internal tunnelDir + cluster name
	if err != nil {
		return nil, err
	}

	var state TunnelState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse tunnel state for %s: %w", clusterName, err)
	}
	// ClusterName is authoritative from the filename; keep it consistent.
	state.ClusterName = clusterName
	return &state, nil
}

// ListActiveTunnels returns all tunnel states from disk
func ListActiveTunnels() ([]TunnelState, error) {
	if err := EnsureTunnelDir(); err != nil {
		return nil, err
	}

	matches, err := filepath.Glob(filepath.Join(tunnelDir, "*.json"))
	if err != nil {
		return nil, err
	}

	var tunnels []TunnelState
	for _, match := range matches {
		name := strings.TrimSuffix(filepath.Base(match), ".json")
		state, err := LoadTunnelState(name)
		if err != nil {
			continue
		}
		tunnels = append(tunnels, *state)
	}
	return tunnels, nil
}

// PruneTunnelDir removes any files in the tunnel directory that are not part of
// the current state format, keeping the directory owned solely by this app. It
// is a best-effort hygiene sweep and never touches running processes.
//
// Files kept:
//   - <cluster>.json           — current per-cluster state files
//   - ssm-<port>.log           — SSM session logs referenced by an active tunnel
//
// Everything else is removed: legacy per-field fragment files from older
// versions (.ssm.pid, .endpoint, ...), stray *.tmp files from an interrupted
// atomic write, and orphaned ssm-*.log files whose port no tunnel references.
func PruneTunnelDir() {
	if err := EnsureTunnelDir(); err != nil {
		return
	}

	entries, err := os.ReadDir(tunnelDir)
	if err != nil {
		return
	}

	// Build the set of log files still referenced by an active tunnel.
	referencedLogs := make(map[string]struct{})
	tunnels, _ := ListActiveTunnels() //nolint:errcheck // best-effort; empty on error
	for _, t := range tunnels {
		if t.LocalPort > 0 {
			referencedLogs[fmt.Sprintf("ssm-%d.log", t.LocalPort)] = struct{}{}
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()

		// Keep current state files.
		if strings.HasSuffix(name, ".json") {
			continue
		}
		// Keep SSM logs that belong to an active tunnel; drop orphaned ones.
		if strings.HasPrefix(name, "ssm-") && strings.HasSuffix(name, ".log") {
			if _, ok := referencedLogs[name]; ok {
				continue
			}
		}
		// Anything else is not part of the current format — remove it.
		_ = os.Remove(filepath.Join(tunnelDir, name)) //nolint:errcheck // best-effort hygiene sweep
	}
}

// IsProcessAlive checks if a process with the given PID is still running
// and is actually one of our managed processes (aws ssm or socat)
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	// Get the command name for this PID
	out, err := exec.CommandContext(context.Background(), "ps", "-p", strconv.Itoa(pid), "-o", "command=").Output() //nolint:gosec // G204 - pid is internally tracked integer
	if err != nil {
		return false
	}
	cmd := strings.TrimSpace(string(out))
	// Verify it's one of our processes
	return strings.Contains(cmd, "aws ssm") ||
		strings.Contains(cmd, "session-manager-plugin") ||
		strings.Contains(cmd, "socat")
}

// KillProcess sends SIGTERM to a process
func KillProcess(pid int, useSudo bool) {
	if pid <= 0 {
		return
	}
	if useSudo {
		_ = exec.CommandContext(context.Background(), "sudo", "kill", strconv.Itoa(pid)).Run() //nolint:errcheck,gosec // best-effort process kill, pid is internally tracked
	} else {
		if p, err := os.FindProcess(pid); err == nil {
			_ = p.Kill() //nolint:errcheck // best-effort process kill
		}
	}
}

// CleanupTunnelState removes the state file (and associated SSM log) for a
// cluster. It also removes any legacy fragment files from older dp-cli versions
// so upgrades don't leave stray files behind.
func CleanupTunnelState(clusterName string) {
	prefix := filepath.Join(tunnelDir, clusterName)

	// Remove the SSM log file, using the port recorded in state if available.
	if state, err := LoadTunnelState(clusterName); err == nil && state.LocalPort > 0 {
		_ = os.Remove(filepath.Join(tunnelDir, fmt.Sprintf("ssm-%d.log", state.LocalPort))) //nolint:errcheck // best-effort log cleanup
	}

	// Remove the JSON state file.
	_ = os.Remove(stateFilePath(clusterName)) //nolint:errcheck // best-effort state cleanup

	// Sweep any legacy per-field fragment files from pre-JSON versions.
	legacyExtensions := []string{".ssm.pid", ".socat.pid", ".endpoint", ".loopback", ".ipv4", ".port", ".nosudo"}
	for _, ext := range legacyExtensions {
		_ = os.Remove(prefix + ext) //nolint:errcheck // best-effort legacy cleanup
	}
}

// CleanupStaleTunnels finds and kills any orphaned socat/SSM processes from
// previous runs. When allowSudo is false (no-sudo mode) it will not perform any
// privileged cleanup of leftover legacy tunnels (killing root-owned socat or
// editing /etc/hosts), so the user is never prompted for a password; such
// tunnels are left in place with their state file intact for a later
// `dp eks session stop`.
func CleanupStaleTunnels(allowSudo bool) {
	tunnels, err := ListActiveTunnels()
	if err != nil {
		return
	}

	for _, t := range tunnels {
		ssmAlive := IsProcessAlive(t.SSMPid)

		// No-sudo tunnels have no socat process and no /etc/hosts entry. Their
		// health depends solely on the SSM port-forward.
		if t.NoSudo {
			if !ssmAlive {
				CleanupTunnelState(t.ClusterName)
			}
			continue
		}

		// Legacy tunnel: privileged cleanup requires sudo. Skip it when not
		// allowed rather than prompting for a password.
		if !allowSudo {
			continue
		}

		socatAlive := IsProcessAlive(t.SocatPid)

		// If both are dead, just clean up state
		if !ssmAlive && !socatAlive {
			if t.Endpoint != "" {
				RemoveHostsEntry(t.Endpoint)
			}
			CleanupTunnelState(t.ClusterName)
			continue
		}

		// If only one is alive, kill it and clean up
		if socatAlive && !ssmAlive {
			KillProcess(t.SocatPid, true)
			if t.Endpoint != "" {
				RemoveHostsEntry(t.Endpoint)
			}
			CleanupTunnelState(t.ClusterName)
		}
		if ssmAlive && !socatAlive {
			KillProcess(t.SSMPid, false)
			if t.Endpoint != "" {
				RemoveHostsEntry(t.Endpoint)
			}
			CleanupTunnelState(t.ClusterName)
		}
	}
}

// CheckAPIConnectivity performs an end-to-end check by opening a TCP connection
// to the EKS API endpoint on port 443. A successful dial confirms the tunnel is
// routing traffic without requiring TLS validation or valid auth credentials.
func CheckAPIConnectivity(endpoint string) bool {
	return checkTCPConnectivity(endpoint + ":443")
}

// CheckLocalConnectivity checks connectivity for a no-sudo tunnel by dialling the
// local SSM port-forward directly (there is no socat/hosts layer to present the
// tunnel on the real endpoint:443).
func CheckLocalConnectivity(localPort int) bool {
	return checkTCPConnectivity(fmt.Sprintf("127.0.0.1:%d", localPort))
}

func checkTCPConnectivity(addr string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

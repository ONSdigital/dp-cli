package eks

import (
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
	tunnelDir   = "/tmp/eks-tunnels-ssm"
	hostsMarker = "# EKS-TUNNEL-MANAGED"
	basePort    = 9443
	maxPort     = 9500
)

// TunnelState represents the persisted state of an active tunnel
type TunnelState struct {
	ClusterName string
	SSMPid      int
	SocatPid    int
	Endpoint    string
	LoopbackIP  string
	IPv4        string
	LocalPort   int
}

// EnsureTunnelDir creates the tunnel state directory if it doesn't exist
func EnsureTunnelDir() error {
	return os.MkdirAll(tunnelDir, 0755)
}

// AllocateLocalPort finds an available port in the range 9443-9500
func AllocateLocalPort() (int, error) {
	for port := basePort; port < maxPort; port++ {
		cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port))
		if err := cmd.Run(); err != nil {
			// lsof returns non-zero if port is not in use
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports in range %d-%d", basePort, maxPort)
}

// AllocateLoopbackIP finds the next available loopback IP where port 443 is not in use.
// Checks 127.0.0.1 through 127.0.0.254.
// Uses sudo lsof to detect root-owned processes (socat runs as root).
func AllocateLoopbackIP() (string, error) {
	for i := 1; i <= 254; i++ {
		ip := fmt.Sprintf("127.0.0.%d", i)
		// Check if port 443 is already bound on this IP (need sudo to see root processes)
		cmd := exec.Command("sudo", "lsof", "-i", fmt.Sprintf("@%s:443", ip))
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
		out, err := exec.Command("ifconfig", "lo0").Output()
		if err != nil {
			return fmt.Errorf("failed to check loopback interfaces: %w", err)
		}
		if strings.Contains(string(out), "inet "+ip+" ") {
			return nil
		}
		cmd := exec.Command("sudo", "ifconfig", "lo0", "alias", ip)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case "linux":
		out, err := exec.Command("ip", "addr", "show", "dev", "lo").Output()
		if err != nil {
			return fmt.Errorf("failed to check loopback interfaces: %w", err)
		}
		if strings.Contains(string(out), " "+ip+"/") {
			return nil
		}
		cmd := exec.Command("sudo", "ip", "addr", "add", ip+"/8", "dev", "lo")
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
		"--parameters", fmt.Sprintf(`{"host":["%s"],"portNumber":["443"],"localPortNumber":["%d"]}`, targetIP, localPort),
	}
	if profile != "" {
		args = append(args, "--profile", profile)
	}

	cmd := exec.Command("aws", args...)

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

	cmd := exec.Command("sudo", "socat", listenAddr, targetAddr)
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
	out, err := exec.Command("pgrep", "-f", pattern).Output()
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

// AddHostsEntry adds an entry to /etc/hosts for the given IP and hostname
func AddHostsEntry(ip, hostname, clusterName string) error {
	// Remove any existing entry first
	RemoveHostsEntry(hostname)

	entry := fmt.Sprintf("%s %s %s %s\n", ip, hostname, hostsMarker, clusterName)
	cmd := exec.Command("sudo", "tee", "-a", "/etc/hosts")
	cmd.Stdin = strings.NewReader(entry)
	cmd.Stdout = nil // suppress tee output
	return cmd.Run()
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
	cmd := exec.Command("sudo", "tee", "/etc/hosts")
	cmd.Stdin = strings.NewReader(strings.Join(filtered, "\n"))
	cmd.Stdout = nil
	cmd.Run()
}

// FlushDNSCache flushes the OS DNS cache. Behaviour is OS-specific and best-effort.
func FlushDNSCache() {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("sudo", "dscacheutil", "-flushcache").Run()
		exec.Command("sudo", "killall", "-HUP", "mDNSResponder").Run()
	case "linux":
		// Try resolvectl (systemd 239+), then systemd-resolve, then nscd — all best-effort
		if exec.Command("resolvectl", "flush-caches").Run() != nil {
			if exec.Command("systemd-resolve", "--flush-caches").Run() != nil {
				exec.Command("sudo", "systemctl", "restart", "nscd").Run()
			}
		}
	}
}

// SaveTunnelState persists tunnel state to disk
func SaveTunnelState(state TunnelState) error {
	prefix := filepath.Join(tunnelDir, state.ClusterName)
	writes := map[string]string{
		prefix + ".ssm.pid":   strconv.Itoa(state.SSMPid),
		prefix + ".socat.pid": strconv.Itoa(state.SocatPid),
		prefix + ".endpoint":  state.Endpoint,
		prefix + ".loopback":  state.LoopbackIP,
		prefix + ".ipv4":      state.IPv4,
		prefix + ".port":      strconv.Itoa(state.LocalPort),
	}
	for path, content := range writes {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
	}
	return nil
}

// LoadTunnelState reads tunnel state from disk
func LoadTunnelState(clusterName string) (*TunnelState, error) {
	prefix := filepath.Join(tunnelDir, clusterName)

	ssmPidBytes, err := os.ReadFile(prefix + ".ssm.pid")
	if err != nil {
		return nil, err
	}
	socatPidBytes, _ := os.ReadFile(prefix + ".socat.pid")
	endpointBytes, _ := os.ReadFile(prefix + ".endpoint")
	loopbackBytes, _ := os.ReadFile(prefix + ".loopback")
	ipv4Bytes, _ := os.ReadFile(prefix + ".ipv4")
	portBytes, _ := os.ReadFile(prefix + ".port")

	ssmPid, _ := strconv.Atoi(strings.TrimSpace(string(ssmPidBytes)))
	socatPid, _ := strconv.Atoi(strings.TrimSpace(string(socatPidBytes)))
	localPort, _ := strconv.Atoi(strings.TrimSpace(string(portBytes)))

	return &TunnelState{
		ClusterName: clusterName,
		SSMPid:      ssmPid,
		SocatPid:    socatPid,
		Endpoint:    strings.TrimSpace(string(endpointBytes)),
		LoopbackIP:  strings.TrimSpace(string(loopbackBytes)),
		IPv4:        strings.TrimSpace(string(ipv4Bytes)),
		LocalPort:   localPort,
	}, nil
}

// ListActiveTunnels returns all tunnel states from disk
func ListActiveTunnels() ([]TunnelState, error) {
	if err := EnsureTunnelDir(); err != nil {
		return nil, err
	}

	matches, err := filepath.Glob(filepath.Join(tunnelDir, "*.ssm.pid"))
	if err != nil {
		return nil, err
	}

	var tunnels []TunnelState
	for _, match := range matches {
		name := strings.TrimSuffix(filepath.Base(match), ".ssm.pid")
		state, err := LoadTunnelState(name)
		if err != nil {
			continue
		}
		tunnels = append(tunnels, *state)
	}
	return tunnels, nil
}

// IsProcessAlive checks if a process with the given PID is still running
// and is actually one of our managed processes (aws ssm or socat)
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	// Get the command name for this PID
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=").Output()
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
		exec.Command("sudo", "kill", strconv.Itoa(pid)).Run()
	} else {
		if p, err := os.FindProcess(pid); err == nil {
			p.Kill()
		}
	}
}

// CleanupTunnelState removes all state files for a cluster
func CleanupTunnelState(clusterName string) {
	prefix := filepath.Join(tunnelDir, clusterName)

	// Read the port before deleting so we can clean up the log file
	portBytes, _ := os.ReadFile(prefix + ".port")
	port := strings.TrimSpace(string(portBytes))
	if port != "" {
		os.Remove(filepath.Join(tunnelDir, fmt.Sprintf("ssm-%s.log", port)))
	}

	extensions := []string{".ssm.pid", ".socat.pid", ".endpoint", ".loopback", ".ipv4", ".port"}
	for _, ext := range extensions {
		os.Remove(prefix + ext)
	}
}

// CleanupStaleTunnels finds and kills any orphaned socat/SSM processes from previous runs
func CleanupStaleTunnels() {
	tunnels, err := ListActiveTunnels()
	if err != nil {
		return
	}

	for _, t := range tunnels {
		ssmAlive := IsProcessAlive(t.SSMPid)
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
	conn, err := net.DialTimeout("tcp", endpoint+":443", 5*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

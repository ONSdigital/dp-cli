package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ONSdigital/dp-cli/aws"
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/eks"
	"github.com/ONSdigital/dp-cli/out"
	"github.com/spf13/cobra"
)

// errSilent is returned to indicate failure without cobra printing the error (SilenceErrors is set).
var errSilent = errors.New("")

func eksCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eks",
		Short: "EKS cluster management commands",
	}

	cmd.AddCommand(eksSessionCommand(ctx, cfg))

	return cmd
}

func eksSessionCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	// Apply optional config overrides for the state dir and local port range so
	// that start, stop and status all operate on the same configured location.
	// Empty/zero values fall back to defaults.
	eks.Configure(cfg.EKS.StateDir, cfg.EKS.BasePort, cfg.EKS.MaxPort)

	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage EKS secure session tunnels",
	}

	cmd.AddCommand(eksSessionStartCommand(ctx, cfg))
	cmd.AddCommand(eksSessionStopCommand(ctx, cfg))
	cmd.AddCommand(eksSessionStatusCommand())

	return cmd
}

func eksSessionStartCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start EKS tunnel sessions for an environment",
	}

	roleFlag := cmd.PersistentFlags().StringP("role", "r", "", "Override role (view, engineer, admin)")
	legacySudoFlag := cmd.PersistentFlags().Bool("legacy-sudo", false, "Use the legacy socat/hosts/loopback tunnel path (requires sudo and socat). Default is no-sudo mode using kubeconfig tls-server-name.")

	// Create a subcommand for each environment
	//nolint:gocritic // rangeValCopy acceptable for environment config iteration
	for _, e := range cfg.Environments {
		env := e
		cmd.AddCommand(&cobra.Command{
			Use:           env.Name,
			Short:         fmt.Sprintf("Start EKS tunnel sessions for %s", env.Name),
			SilenceUsage:  true,
			SilenceErrors: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				return runSessionStart(ctx, cfg, env, *roleFlag, *legacySudoFlag)
			},
		})
	}

	return cmd
}

func eksSessionStopCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop [environment]",
		Short: "Stop EKS tunnel sessions (all if no environment specified)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return runSessionStopEnv(args[0])
			}
			runSessionStopAll() //nolint:errcheck // best-effort cleanup on signal
			return nil
		},
	}

	return cmd
}

func eksSessionStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show active EKS tunnel sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSessionStatus()
		},
	}
}

func runSessionStart(ctx context.Context, cfg *config.Config, env config.Environment, roleOverride string, legacySudoFlag bool) error {
	// No-sudo mode is the default. Opt in to the legacy socat/hosts/loopback path
	// with --legacy-sudo. No sudo probe is performed here (it added latency and
	// was unstable), so the common path never touches sudo at all.
	noSudo := eks.ResolveNoSudoMode(legacySudoFlag)

	// Check dependencies (socat only required for the legacy sudo path)
	missing := eks.CheckDependencies(noSudo)
	if len(missing) > 0 {
		out.ErrorFHighlight("Missing required dependencies:")
		for _, dep := range missing {
			out.ErrorFHighlight("  ✗ %s (%s) - %s", dep.Name, dep.Command, dep.InstallHint)
		}
		out.ErrorFHighlight("Install missing dependencies before continuing")
		return errSilent
	}

	// Resolve profile based on role override or command default
	var profile string
	var roleSuffix string
	if roleOverride != "" {
		profile = cfg.GetProfileWithRole(env.Name, roleOverride)
		roleSuffix = cfg.GetRoleSuffix(roleOverride)
	} else {
		// Get the AWS profile for this environment and command
		profile = cfg.GetProfileForCommand(env.Name, "eks.session.start")
		// Determine which role was resolved for the suffix
		if profile != cfg.GetProfile(env.Name) {
			// Find the suffix that was applied
			base := cfg.GetProfile(env.Name)
			roleSuffix = strings.TrimPrefix(profile, base)
		} else {
			roleSuffix = cfg.GetRoleSuffix("view")
			if roleSuffix == "" {
				roleSuffix = ""
			}
		}
	}

	tags := eks.NewDiscoveryTags(cfg.EKS.TunnelBoxRoleTag, cfg.EKS.ClusterAccessTag)

	out.InfoFHighlight("Starting EKS session for environment: %s (profile: %s)", env.Name, profile)

	// Discover tunnel box
	out.Info("  Discovering tunnel box...")
	tunnelBox, err := eks.FindTunnelBox(ctx, profile, tags)
	if err != nil {
		out.ErrorFHighlight("  %s Tunnel box discovery failed", "✗")
		out.ErrorFHighlight("  %s", err)
		if aws.IsAccessError(err) {
			out.AccessDeniedGuidance(profile)
		}
		return errSilent
	}
	out.InfoFHighlight("  ✓ Found tunnel box: %s (%s)", tunnelBox.Name, tunnelBox.InstanceID)

	// Discover clusters
	out.Info("  Discovering EKS clusters...")
	clusters, err := eks.FindClusters(ctx, profile, tags)
	if err != nil {
		out.ErrorFHighlight("  %s Cluster discovery failed", "✗")
		out.ErrorFHighlight("  %s", err)
		if aws.IsAccessError(err) {
			out.AccessDeniedGuidance(profile)
		}
		return errSilent
	}
	out.InfoFHighlight("  ✓ Found %s cluster(s)", fmt.Sprintf("%d", len(clusters)))
	for _, c := range clusters {
		out.InfoFHighlight("    - %s", c.Name)
	}

	// Setup signal handler for graceful cleanup
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		out.Warn("\n  Caught interrupt, cleaning up...")
		_ = runSessionStopAll() //nolint:errcheck // best-effort cleanup on signal
		os.Exit(0)
	}()

	// Ensure tunnel directory exists
	if err := eks.EnsureTunnelDir(); err != nil {
		return fmt.Errorf("failed to create tunnel directory: %w", err)
	}

	// Hygiene: remove anything in the tunnel dir that isn't part of the current
	// state format (legacy fragment files from older versions, stray temp files,
	// orphaned logs). Keeps the dir owned solely by this app.
	eks.PruneTunnelDir()

	if !noSudo {
		out.WarnFHighlight("  Using legacy sudo tunnel mode (socat, /etc/hosts, and loopback alias). You may be prompted for your password.")
		// Ensure sudo credentials are cached before any privileged operations
		if err := eks.EnsureSudo("binding port 443 (socat), updating /etc/hosts, and creating loopback aliases"); err != nil {
			return err
		}
	}

	// Clean up any stale tunnels before starting
	eks.CleanupStaleTunnels(!noSudo)

	// Start tunnels for each cluster
	tunnelsEstablished := 0
	var accessDenied bool
	for _, cluster := range clusters {
		var loopbackIP string
		if !noSudo {
			// Allocate an available loopback IP (legacy socat path only)
			ip, err := eks.AllocateLoopbackIP()
			if err != nil {
				out.WarnFHighlight("    ⚠ %s", err.Error())
				continue
			}
			loopbackIP = ip
		}

		out.InfoFHighlight("  Setting up tunnel for: %s", cluster.Name)

		ok, accessErr := setupClusterTunnel(ctx, profile, tunnelBox.InstanceID, cluster, loopbackIP, noSudo)
		if accessErr {
			accessDenied = true
		}
		if ok {
			tunnelsEstablished++
		}
	}

	// Flush DNS cache (only relevant for the legacy /etc/hosts path)
	if !noSudo {
		eks.FlushDNSCache()
	}

	// Only update kubeconfig and show success if tunnels were established in this run
	if tunnelsEstablished == 0 {
		out.ErrorFHighlight("  %s No tunnels established", "✗")
		if accessDenied {
			out.AccessDeniedGuidance(profile)
		}
		return errSilent
	}

	// Update kubeconfig for each cluster with an active tunnel
	out.InfoFHighlight("  Updating kubeconfig (profile: %s, user-alias: *%s)...", profile, roleSuffix)
	updateKubeconfigs(clusters, profile, roleSuffix, noSudo)

	out.Info("  ✓ All tunnels active. kubectl, Terraform, and k9s can route through the tunnel box.")
	out.Info("  All access is auditable via CloudTrail SSM session logs.")
	out.Info("")
	out.Info("  Run 'dp eks session status' to check tunnel health.")
	out.Info("  Run 'dp eks session stop' to tear down all tunnels.")

	return nil
}

// updateKubeconfigs runs aws eks update-kubeconfig for each cluster and, in
// no-sudo mode, patches each cluster block to point at the local SSM
// port-forward with tls-server-name set to the real endpoint.
func updateKubeconfigs(clusters []eks.ClusterInfo, profile, roleSuffix string, noSudo bool) {
	for _, cluster := range clusters {
		msg, err := eks.UpdateKubeconfig(cluster.Name, profile, roleSuffix)
		if err != nil {
			out.WarnFHighlight("    ⚠ Failed to configure %s: %s", cluster.Name, err.Error())
			continue
		}
		out.InfoFHighlight("    ✓ %s", msg)

		if !noSudo {
			continue
		}
		// Repoint the cluster at the local SSM port-forward and set the real
		// endpoint as the TLS SNI so kubectl connects to 127.0.0.1:<port>
		// while still fully validating the server certificate.
		state, loadErr := eks.LoadTunnelState(cluster.Name)
		if loadErr != nil {
			out.WarnFHighlight("    ⚠ Could not load tunnel state to patch kubeconfig for %s: %s", cluster.Name, loadErr.Error())
			continue
		}
		if err := eks.PatchKubeconfigForNoSudo(cluster.Name, cluster.Endpoint, state.LocalPort); err != nil {
			out.WarnFHighlight("    ⚠ Failed to patch kubeconfig for %s: %s", cluster.Name, err.Error())
			continue
		}
		out.InfoFHighlight("    ✓ Pointed %s at 127.0.0.1:%s (tls-server-name: %s)", cluster.Name, fmt.Sprintf("%d", state.LocalPort), cluster.Endpoint)
	}
}

func runSessionStopEnv(environment string) error {
	tunnels, err := eks.ListActiveTunnels()
	if err != nil {
		return err
	}

	found := false
	sudoEnsured := false
	for _, t := range tunnels {
		if strings.Contains(t.ClusterName, environment) {
			if !t.NoSudo && !sudoEnsured {
				if err := eks.EnsureSudo("killing socat processes and cleaning /etc/hosts"); err != nil {
					return err
				}
				sudoEnsured = true
			}
			stopTunnel(t, true)
			found = true
		}
	}

	if !found {
		out.WarnFHighlight("  No active tunnels found for environment: %s", environment)
	}

	// DNS flush is only relevant to the legacy /etc/hosts path and itself needs
	// sudo on macOS; only flush if we actually stopped a legacy tunnel.
	if sudoEnsured {
		eks.FlushDNSCache()
	}
	return nil
}

func runSessionStopAll() error {
	tunnels, err := eks.ListActiveTunnels()
	if err != nil {
		return err
	}

	if len(tunnels) == 0 {
		out.Info("  No active tunnels")
		return nil
	}

	// Only need sudo if any tunnel used the legacy socat/hosts path.
	needsSudo := false
	for _, t := range tunnels {
		if !t.NoSudo {
			needsSudo = true
			break
		}
	}
	if needsSudo {
		if err := eks.EnsureSudo("killing socat processes and cleaning /etc/hosts"); err != nil {
			return err
		}
	}

	out.Info("Stopping all EKS tunnels...")
	for _, t := range tunnels {
		stopTunnel(t, true)
	}

	// DNS cache only needs flushing for the legacy /etc/hosts path; flushing on
	// macOS itself requires sudo, so skip it when only no-sudo tunnels were
	// stopped (avoids an otherwise-pointless password prompt).
	if needsSudo {
		eks.FlushDNSCache()
	}
	out.Info("  ✓ All tunnels stopped")
	return nil
}

// stopTunnel tears down a tunnel. allowSudo must be true to perform privileged
// cleanup of a legacy tunnel (killing root-owned socat, editing /etc/hosts).
// When allowSudo is false (a no-sudo start cleaning up leftover legacy state), we
// skip the privileged steps and just remove our own SSM process and state files,
// so the user is never prompted for a password.
func stopTunnel(t eks.TunnelState, allowSudo bool) {
	out.InfoFHighlight("  Stopping tunnel: %s", t.ClusterName)

	if !t.NoSudo && allowSudo && eks.IsProcessAlive(t.SocatPid) {
		eks.KillProcess(t.SocatPid, true)
		out.InfoFHighlight("    Killed socat (PID: %s)", fmt.Sprintf("%d", t.SocatPid))
	}
	if eks.IsProcessAlive(t.SSMPid) {
		eks.KillProcess(t.SSMPid, false)
		out.InfoFHighlight("    Killed SSM session (PID: %s)", fmt.Sprintf("%d", t.SSMPid))
	}
	if !t.NoSudo && allowSudo && t.Endpoint != "" {
		eks.RemoveHostsEntry(t.Endpoint)
		out.Info("    Removed /etc/hosts entry")
	}
	if !t.NoSudo && !allowSudo {
		out.WarnFHighlight("    ⚠ Leftover tunnel for %s needs elevated cleanup. Run 'dp eks session stop' to fully clean up.", t.ClusterName)
	}
	eks.CleanupTunnelState(t.ClusterName)
}

func runSessionStatus() error {
	tunnels, err := eks.ListActiveTunnels()
	if err != nil {
		return err
	}

	if len(tunnels) == 0 {
		out.Info("  No active EKS tunnels")
		return nil
	}

	out.Info("Checking EKS tunnel status...")

	for _, t := range tunnels {
		ssmAlive := eks.IsProcessAlive(t.SSMPid)

		ssmStatus := "RUNNING"
		if !ssmAlive {
			ssmStatus = "DEAD"
		}

		out.InfoFHighlight("  Cluster:  %s", t.ClusterName)
		if ssmAlive {
			out.InfoFHighlight("    SSM:      %s (PID: %s, port: %s)", ssmStatus, fmt.Sprintf("%d", t.SSMPid), fmt.Sprintf("%d", t.LocalPort))
		} else {
			out.ErrorFHighlight("    SSM:      %s (PID: %s, port: %s)", ssmStatus, fmt.Sprintf("%d", t.SSMPid), fmt.Sprintf("%d", t.LocalPort))
		}

		if !t.NoSudo {
			socatAlive := eks.IsProcessAlive(t.SocatPid)
			socatStatus := "RUNNING"
			if !socatAlive {
				socatStatus = "DEAD"
			}
			if socatAlive {
				out.InfoFHighlight("    socat:    %s (PID: %s, %s:443)", socatStatus, fmt.Sprintf("%d", t.SocatPid), t.LoopbackIP)
			} else {
				out.ErrorFHighlight("    socat:    %s (PID: %s, %s:443)", socatStatus, fmt.Sprintf("%d", t.SocatPid), t.LoopbackIP)
			}
		}
		out.InfoFHighlight("    Endpoint: %s", t.Endpoint)

		// End-to-end connectivity check
		apiStatus := "SKIPPED"
		if t.NoSudo {
			if ssmAlive {
				if eks.CheckLocalConnectivity(t.LocalPort) {
					out.InfoFHighlight("    API:      %s", "REACHABLE")
				} else {
					out.ErrorFHighlight("    API:      %s (tunnel may be stale)", "UNREACHABLE")
				}
			} else {
				out.WarnFHighlight("    API:      %s (SSM session not healthy)", apiStatus)
			}
			out.InfoFHighlight("    Route:    kubectl → 127.0.0.1:%s → tunnel box → EKS", fmt.Sprintf("%d", t.LocalPort))
			continue
		}

		socatAlive := eks.IsProcessAlive(t.SocatPid)
		if ssmAlive && socatAlive && t.Endpoint != "" {
			apiReachable := eks.CheckAPIConnectivity(t.Endpoint)
			if apiReachable {
				apiStatus = "REACHABLE"
			} else {
				apiStatus = "UNREACHABLE"
			}
			if apiReachable {
				out.InfoFHighlight("    API:      %s", apiStatus)
			} else {
				out.ErrorFHighlight("    API:      %s (tunnel may be stale)", apiStatus)
			}
		} else {
			out.WarnFHighlight("    API:      %s (processes not healthy)", apiStatus)
		}

		out.InfoFHighlight("    Route:    %s → %s:443 → 127.0.0.1:%s → tunnel box → EKS", t.Endpoint, t.LoopbackIP, fmt.Sprintf("%d", t.LocalPort))
	}

	return nil
}

// setupClusterTunnel handles the tunnel setup for a single cluster.
// Returns (success bool, accessDenied bool).
func setupClusterTunnel(ctx context.Context, profile, tunnelBoxID string, cluster eks.ClusterInfo, loopbackIP string, noSudo bool) (success, accessDenied bool) {
	// Check if tunnel already running and healthy. In no-sudo mode there is no
	// socat process, so health depends solely on the SSM port-forward.
	existing, err := eks.LoadTunnelState(cluster.Name)
	if err == nil {
		healthy := eks.IsProcessAlive(existing.SSMPid)
		if !existing.NoSudo {
			healthy = healthy && eks.IsProcessAlive(existing.SocatPid)
		}
		if healthy {
			out.InfoFHighlight("    Tunnel already active (SSM PID: %s)", fmt.Sprintf("%d", existing.SSMPid))
			return true, false
		}
	}
	// If state exists but processes are dead, clean up first. In no-sudo mode we
	// must not perform privileged cleanup of a leftover legacy tunnel (that would
	// prompt for a password), so pass allowSudo=!noSudo.
	if err == nil {
		stopTunnel(*existing, !noSudo)
	}

	// Resolve IPv4 via tunnel box (required for dualstack clusters to force A record)
	out.Info("    Resolving endpoint IPv4 via tunnel box...")
	ipv4, err := eks.ResolveEndpointIPv4(ctx, profile, tunnelBoxID, cluster.Endpoint)
	if err != nil {
		out.ErrorFHighlight("  %s Failed to resolve cluster: %s", "✗", cluster.Name)
		out.ErrorFHighlight("  %s", err.Error())
		return false, aws.IsAccessError(err)
	}
	out.InfoFHighlight("    IPv4: %s", ipv4)

	// Allocate a local port and hold it reserved until just before the SSM child
	// binds, so a concurrent/sequential multi-cluster start can't hand the same
	// port to two clusters.
	localPort, portHold, err := eks.AllocateAndHoldLocalPort()
	if err != nil {
		out.WarnFHighlight("    ⚠ %s", err.Error())
		return false, false
	}
	out.InfoFHighlight("    Local port: %s", fmt.Sprintf("%d", localPort))

	// Release the reservation immediately before the SSM session binds the port.
	_ = portHold.Close() //nolint:errcheck // releasing our port reservation before handoff

	// Start SSM port forward
	out.Info("    Starting SSM session...")
	ssmPid, err := eks.StartSSMPortForward(tunnelBoxID, ipv4, localPort, profile)
	if err != nil {
		out.WarnFHighlight("    ⚠ SSM session failed: %s", err.Error())
		return false, false
	}
	out.InfoFHighlight("    SSM session established (PID: %s)", fmt.Sprintf("%d", ssmPid))

	if noSudo {
		// No-sudo path: no loopback alias, no socat, no /etc/hosts. kubectl will
		// be pointed at 127.0.0.1:localPort via a patched kubeconfig after this.
		state := eks.TunnelState{
			ClusterName: cluster.Name,
			SSMPid:      ssmPid,
			Endpoint:    cluster.Endpoint,
			IPv4:        ipv4,
			LocalPort:   localPort,
			NoSudo:      true,
		}
		if err := eks.SaveTunnelState(state); err != nil {
			out.WarnFHighlight("    ⚠ Failed to save tunnel state: %s", err.Error())
		}
		out.InfoFHighlight("    ✓ Tunnel active: 127.0.0.1:%s → tunnel box → %s:443", fmt.Sprintf("%d", localPort), ipv4)
		return true, false
	}

	// Ensure loopback alias
	if err := eks.EnsureLoopbackAlias(loopbackIP); err != nil {
		out.WarnFHighlight("    ⚠ Failed to create loopback alias: %s", err.Error())
		eks.KillProcess(ssmPid, false)
		return false, false
	}

	// Start socat
	out.InfoFHighlight("    Starting socat (%s:443 → 127.0.0.1:%s)...", loopbackIP, fmt.Sprintf("%d", localPort))
	socatPid, err := eks.StartSocat(loopbackIP, localPort)
	if err != nil {
		out.WarnFHighlight("    ⚠ socat failed: %s", err.Error())
		eks.KillProcess(ssmPid, false)
		return false, false
	}

	// Add hosts entry
	if err := eks.AddHostsEntry(loopbackIP, cluster.Endpoint, cluster.Name); err != nil {
		out.WarnFHighlight("    ⚠ Failed to update /etc/hosts: %s", err.Error())
		eks.KillProcess(ssmPid, false)
		eks.KillProcess(socatPid, true)
		return false, false
	}

	// Save state
	state := eks.TunnelState{
		ClusterName: cluster.Name,
		SSMPid:      ssmPid,
		SocatPid:    socatPid,
		Endpoint:    cluster.Endpoint,
		LoopbackIP:  loopbackIP,
		IPv4:        ipv4,
		LocalPort:   localPort,
	}
	if err := eks.SaveTunnelState(state); err != nil {
		out.WarnFHighlight("    ⚠ Failed to save tunnel state: %s", err.Error())
	}

	out.InfoFHighlight("    ✓ Tunnel active: %s → %s:443 → tunnel box → %s:443", cluster.Endpoint, loopbackIP, ipv4)
	return true, false
}

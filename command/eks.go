package command

import (
	"context"
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

func eksCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eks",
		Short: "EKS cluster management commands",
	}

	cmd.AddCommand(eksSessionCommand(ctx, cfg))

	return cmd
}

func eksSessionCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
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

	// Create a subcommand for each environment
	//nolint:gocritic // rangeValCopy acceptable for environment config iteration
	for _, e := range cfg.Environments {
		env := e
		cmd.AddCommand(&cobra.Command{
			Use:   env.Name,
			Short: fmt.Sprintf("Start EKS tunnel sessions for %s", env.Name),
			RunE: func(cmd *cobra.Command, args []string) error {
				return runSessionStart(ctx, cfg, env, *roleFlag)
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
			return runSessionStopAll()
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

func runSessionStart(ctx context.Context, cfg *config.Config, env config.Environment, roleOverride string) error {
	// Check dependencies
	missing := eks.CheckDependencies()
	if len(missing) > 0 {
		out.ErrorFHighlight("Missing required dependencies:")
		for _, dep := range missing {
			out.ErrorFHighlight("  ✗ %s (%s) - %s", dep.Name, dep.Command, dep.InstallHint)
		}
		return fmt.Errorf("install missing dependencies before continuing")
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
		return nil
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
		return nil
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
		runSessionStopAll()
		os.Exit(0)
	}()

	// Ensure tunnel directory exists
	if err := eks.EnsureTunnelDir(); err != nil {
		return fmt.Errorf("failed to create tunnel directory: %w", err)
	}

	// Ensure sudo credentials are cached before any privileged operations
	if err := eks.EnsureSudo("binding port 443 (socat), updating /etc/hosts, and creating loopback aliases"); err != nil {
		return err
	}

	// Clean up any stale tunnels before starting
	eks.CleanupStaleTunnels()

	// Start tunnels for each cluster
	tunnelsEstablished := 0
	var accessDenied bool
	for _, cluster := range clusters {
		// Allocate an available loopback IP
		loopbackIP, err := eks.AllocateLoopbackIP()
		if err != nil {
			out.WarnFHighlight("    ⚠ %s", err.Error())
			continue
		}

		out.InfoFHighlight("  Setting up tunnel for: %s", cluster.Name)

		// Check if tunnel already running and healthy - skip tunnel setup
		existing, err := eks.LoadTunnelState(cluster.Name)
		if err == nil && eks.IsProcessAlive(existing.SSMPid) && eks.IsProcessAlive(existing.SocatPid) {
			out.InfoFHighlight("    Tunnel already active (SSM PID: %s)", fmt.Sprintf("%d", existing.SSMPid))
			tunnelsEstablished++
			continue
		}
		// If state exists but processes are dead, clean up first
		if err == nil {
			stopTunnel(*existing)
		}

		// Resolve IPv4 via tunnel box (required for dualstack clusters to force A record)
		out.Info("    Resolving endpoint IPv4 via tunnel box...")
		ipv4, err := eks.ResolveEndpointIPv4(ctx, profile, tunnelBox.InstanceID, cluster.Endpoint)
		if err != nil {
			out.ErrorFHighlight("  %s Failed to resolve cluster: %s", "✗", cluster.Name)
			out.ErrorFHighlight("  %s", err.Error())
			if aws.IsAccessError(err) {
				accessDenied = true
			}
			continue
		}
		out.InfoFHighlight("    IPv4: %s", ipv4)

		// Allocate local port
		localPort, err := eks.AllocateLocalPort()
		if err != nil {
			out.WarnFHighlight("    ⚠ %s", err.Error())
			continue
		}
		out.InfoFHighlight("    Local port: %s", fmt.Sprintf("%d", localPort))

		// Start SSM port forward
		out.Info("    Starting SSM session...")
		ssmPid, err := eks.StartSSMPortForward(tunnelBox.InstanceID, ipv4, localPort, profile)
		if err != nil {
			out.WarnFHighlight("    ⚠ SSM session failed: %s", err.Error())
			continue
		}
		out.InfoFHighlight("    SSM session established (PID: %s)", fmt.Sprintf("%d", ssmPid))

		// Ensure loopback alias
		if err := eks.EnsureLoopbackAlias(loopbackIP); err != nil {
			out.WarnFHighlight("    ⚠ Failed to create loopback alias: %s", err.Error())
			eks.KillProcess(ssmPid, false)
			continue
		}

		// Start socat
		out.InfoFHighlight("    Starting socat (%s:443 → 127.0.0.1:%s)...", loopbackIP, fmt.Sprintf("%d", localPort))
		socatPid, err := eks.StartSocat(loopbackIP, localPort)
		if err != nil {
			out.WarnFHighlight("    ⚠ socat failed: %s", err.Error())
			eks.KillProcess(ssmPid, false)
			continue
		}

		// Add hosts entry
		if err := eks.AddHostsEntry(loopbackIP, cluster.Endpoint, cluster.Name); err != nil {
			out.WarnFHighlight("    ⚠ Failed to update /etc/hosts: %s", err.Error())
			eks.KillProcess(ssmPid, false)
			eks.KillProcess(socatPid, true)
			continue
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
		tunnelsEstablished++
	}

	// Flush DNS cache
	eks.FlushDNSCache()

	// Only update kubeconfig and show success if tunnels were established in this run
	if tunnelsEstablished == 0 {
		out.ErrorFHighlight("  %s No tunnels established", "✗")
		if accessDenied {
			out.AccessDeniedGuidance(profile)
		}
		return nil
	}

	// Update kubeconfig for each cluster with an active tunnel
	out.InfoFHighlight("  Updating kubeconfig (profile: %s, user-alias: *%s)...", profile, roleSuffix)
	for _, cluster := range clusters {
		msg, err := eks.UpdateKubeconfig(cluster.Name, profile, roleSuffix)
		if err != nil {
			out.WarnFHighlight("    ⚠ Failed to configure %s: %s", cluster.Name, err.Error())
			continue
		}
		out.InfoFHighlight("    ✓ %s", msg)
	}

	out.Info("  ✓ All tunnels active. kubectl, Terraform, and k9s can route through the tunnel box.")
	out.Info("  All access is auditable via CloudTrail SSM session logs.")
	out.Info("")
	out.Info("  Run 'dp eks session status' to check tunnel health.")
	out.Info("  Run 'dp eks session stop' to tear down all tunnels.")

	return nil
}

func runSessionStopEnv(environment string) error {
	tunnels, err := eks.ListActiveTunnels()
	if err != nil {
		return err
	}

	found := false
	for _, t := range tunnels {
		if strings.Contains(t.ClusterName, environment) {
			if !found {
				if err := eks.EnsureSudo("killing socat processes and cleaning /etc/hosts"); err != nil {
					return err
				}
			}
			stopTunnel(t)
			found = true
		}
	}

	if !found {
		out.WarnFHighlight("  No active tunnels found for environment: %s", environment)
	}

	eks.FlushDNSCache()
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

	if err := eks.EnsureSudo("killing socat processes and cleaning /etc/hosts"); err != nil {
		return err
	}

	out.Info("Stopping all EKS tunnels...")
	for _, t := range tunnels {
		stopTunnel(t)
	}

	eks.FlushDNSCache()
	out.Info("  ✓ All tunnels stopped")
	return nil
}

func stopTunnel(t eks.TunnelState) {
	out.InfoFHighlight("  Stopping tunnel: %s", t.ClusterName)

	if eks.IsProcessAlive(t.SocatPid) {
		eks.KillProcess(t.SocatPid, true)
		out.InfoFHighlight("    Killed socat (PID: %s)", fmt.Sprintf("%d", t.SocatPid))
	}
	if eks.IsProcessAlive(t.SSMPid) {
		eks.KillProcess(t.SSMPid, false)
		out.InfoFHighlight("    Killed SSM session (PID: %s)", fmt.Sprintf("%d", t.SSMPid))
	}
	if t.Endpoint != "" {
		eks.RemoveHostsEntry(t.Endpoint)
		out.Info("    Removed /etc/hosts entry")
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
		socatAlive := eks.IsProcessAlive(t.SocatPid)

		ssmStatus := "RUNNING"
		socatStatus := "RUNNING"
		if !ssmAlive {
			ssmStatus = "DEAD"
		}
		if !socatAlive {
			socatStatus = "DEAD"
		}

		out.InfoFHighlight("  Cluster:  %s", t.ClusterName)
		if ssmAlive {
			out.InfoFHighlight("    SSM:      %s (PID: %s, port: %s)", ssmStatus, fmt.Sprintf("%d", t.SSMPid), fmt.Sprintf("%d", t.LocalPort))
		} else {
			out.ErrorFHighlight("    SSM:      %s (PID: %s, port: %s)", ssmStatus, fmt.Sprintf("%d", t.SSMPid), fmt.Sprintf("%d", t.LocalPort))
		}
		if socatAlive {
			out.InfoFHighlight("    socat:    %s (PID: %s, %s:443)", socatStatus, fmt.Sprintf("%d", t.SocatPid), t.LoopbackIP)
		} else {
			out.ErrorFHighlight("    socat:    %s (PID: %s, %s:443)", socatStatus, fmt.Sprintf("%d", t.SocatPid), t.LoopbackIP)
		}
		out.InfoFHighlight("    Endpoint: %s", t.Endpoint)

		// End-to-end connectivity check
		apiStatus := "SKIPPED"
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

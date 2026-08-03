package command

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ONSdigital/dp-cli/aws"
	"github.com/ONSdigital/dp-cli/cli"
	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"
	"github.com/spf13/cobra"
)

// buildRemoteAccessPayload collects user/IPs from cfg and builds the JSON payload.
// action must be one of: "add", "revoke".
func buildRemoteAccessPayload(cfg *config.Config, action string) ([]byte, error) {
	if cfg.UserName == nil || *cfg.UserName == "" {
		return nil, fmt.Errorf("no user provided (use --user)")
	}

	// If explicit IPs were provided, use only those (avoid external lookup)
	ipv4 := ""
	if cfg.IPv4Address != nil && *cfg.IPv4Address != "" {
		ipv4 = *cfg.IPv4Address
	}
	ipv6 := ""
	if cfg.IPv6Address != nil && *cfg.IPv6Address != "" {
		ipv6 = *cfg.IPv6Address
	}

	// Validate any explicitly-provided IPs
	if ipv4 != "" && !config.IsValidIPv4(ipv4) {
		return nil, fmt.Errorf("invalid IPv4 address: %q", ipv4)
	}
	if ipv6 != "" && !config.IsValidIPv6(ipv6) {
		return nil, fmt.Errorf("invalid IPv6 address: %q", ipv6)
	}

	if ipv4 == "" && ipv6 == "" {
		var err error
		ipv4, ipv6, err = cfg.GetMyIPs()
		if err != nil {
			return nil, err
		}
	}

	ips := make([]string, 0)
	if ipv4 != "" {
		ips = append(ips, ipv4)
	}
	if ipv6 != "" {
		ips = append(ips, ipv6)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no IPv4 or IPv6 address resolved")
	}

	payload := map[string]interface{}{
		"action": action,
		"user":   *cfg.UserName,
		"ips":    ips,
	}

	return json.Marshal(payload)
}

// LambdaResult represents a single item from the Lambda response array
type LambdaResult struct {
	Status             string `json:"status"`
	Message            string `json:"message"`
	ResourceID         string `json:"resource_id"`
	ResourceType       string `json:"resource_type"`
	Action             string `json:"action"`
	TableUpdated       bool   `json:"table_updated"`
	SGGroupName        string `json:"sg_group_name,omitempty"`
	CloudflareListName string `json:"cloudflare_list_name,omitempty"`
	IPVersion          string `json:"ip_version,omitempty"`
	ExpiresAt          *int64 `json:"expires_at,omitempty"`
	RevokedAt          *int64 `json:"revoked_at,omitempty"`
}

func formatUnix(ts int64) string {
	// dd/mm/yyyy hh:mm:ss
	return time.Unix(ts, 0).Local().Format("02/01/2006 15:04:05")
}

//nolint:gocritic // paramTypeCombine - keeping params explicit for readability
func renderLambdaResults(lvl out.Level, envName string, user string, results []LambdaResult) {
	//nolint:gocritic // rangeValCopy acceptable for result iteration
	for _, r := range results {
		verb := "allowing"
		if r.Action == "revoke" {
			verb = "denying"
		}

		// Resource label: prefer SG group name for SGs, otherwise type(id)
		label := r.ResourceType
		switch r.ResourceType {
		case "sg":
			if r.SGGroupName != "" {
				label = fmt.Sprintf("sg %s (%s)", r.SGGroupName, r.ResourceID)
			} else {
				label = fmt.Sprintf("sg (%s)", r.ResourceID)
			}
		case "cloudflare":
			if r.CloudflareListName != "" {
				label = fmt.Sprintf("cloudflare %s (%s)", r.CloudflareListName, r.ResourceID)
			} else {
				label = fmt.Sprintf("cloudflare (%s)", r.ResourceID)
			}
		default:
			if r.ResourceID != "" {
				label = fmt.Sprintf("%s (%s)", r.ResourceType, r.ResourceID)
			}
		}

		// Optional timestamp suffix
		suffix := ""
		if r.ExpiresAt != nil {
			suffix = fmt.Sprintf(" (expires: %s)", formatUnix(*r.ExpiresAt))
		} else if r.RevokedAt != nil {
			suffix = fmt.Sprintf(" (revoked: %s)", formatUnix(*r.RevokedAt))
		}

		// Compose line similar to existing output, using the Lambda message
		// Example: [dp] allowing bob via sandbox - remote-allow-test-1 (sg-...) <message> (expires: ...)
		out.Highlight(lvl, "%s %s via %s - %s %s%s", verb, user, envName, label, r.Message, suffix)
	}
}

// remoteAccess creates the remote command for the remote allow service.
func remoteAccess(ctx context.Context, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remote",
		Short: "Allow or deny remote access to environment",
	}
	ipv4Default := ""
	if cfg.IPv4Address != nil {
		ipv4Default = *cfg.IPv4Address
	}
	ipv4Flag := cmd.PersistentFlags().String("ipv4", ipv4Default, "The IPv4 address for remote sub-commands")
	if ipv4Flag != nil {
		cfg.IPv4Address = ipv4Flag
	}

	ipv6Default := ""
	if cfg.IPv6Address != nil {
		ipv6Default = *cfg.IPv6Address
	}
	ipv6Flag := cmd.PersistentFlags().String("ipv6", ipv6Default, "The IPv6 address for remote sub-commands")
	if ipv6Flag != nil {
		cfg.IPv6Address = ipv6Flag
	}

	userDefault := ""
	if cfg.UserName != nil {
		userDefault = *cfg.UserName
	}
	userFlag := cmd.PersistentFlags().String("user", userDefault, "The user for access lists")
	if userFlag != nil {
		cfg.UserName = userFlag
	}

	cmd.AddCommand(remoteAllowCommand(ctx, cfg))
	cmd.AddCommand(remoteDenyCommand(ctx, cfg))
	cmd.AddCommand(remoteLoginCommand(ctx, cfg))

	return cmd
}

// remoteAllowCommand creates the allow subcommand for remote
func remoteAllowCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "allow",
		Short: "allow access to environment",
	}

	envSubCmds := make([]*cobra.Command, 0, len(cfg.Environments))

	// create subcommands for each environment from the config
	//nolint:gocritic // rangeValCopy acceptable for environment config iteration
	for _, e := range cfg.Environments { //nolint:gocritic // rangeValCopy acceptable for environment config iteration
		env := e
		envSubCmds = append(envSubCmds, &cobra.Command{
			Use:   e.Name,
			Short: "allow access to " + env.Name,
			RunE: func(cmd *cobra.Command, args []string) error {
				lvl := out.GetLevel(env)

				// Build payload
				payload, err := buildRemoteAccessPayload(cfg, "add")
				if err != nil {
					out.ErrorFHighlight("  %s %v", "✗", err)
					return nil
				}

				// Lambda function name pattern: dis-<env>-remote-allow
				functionName := fmt.Sprintf("dis-%s-remote-allow", env.Name)
				// Get the AWS profile for this environment and command
				profile := cfg.GetProfileForCommand(env.Name, "remote.allow")

				out.Highlight(lvl, "invoking lambda %s for %s (profile: %s)", functionName, env.Name, profile)
				resp, err := aws.InvokeLambda(ctx, profile, functionName, payload)
				if err != nil {
					return fmt.Errorf("lambda invoke failed: %w", err)
				}
				// Parse and render response as highlighted lines
				var results []LambdaResult
				if err := json.Unmarshal([]byte(resp), &results); err != nil {
					// Fallback to raw output if not JSON array
					cmd.Printf("%s\n", resp)
					return nil
				}
				user := ""
				if cfg.UserName != nil {
					user = *cfg.UserName
				}
				renderLambdaResults(lvl, env.Name, user, results)
				return nil
			},
		})
	}

	if len(envSubCmds) == 0 {
		out.Warn("Warning: No subcommands found for envs - missing envs in config?")
	}

	cmd.AddCommand(envSubCmds...)
	return cmd
}

// remoteDenyCommand creates the deny subcommand for remote
func remoteDenyCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deny",
		Short: "deny access to environment",
	}

	envSubCmds := make([]*cobra.Command, 0, len(cfg.Environments))

	//nolint:gocritic // rangeValCopy acceptable for environment config iteration
	for _, e := range cfg.Environments {
		env := e
		envSubCmds = append(envSubCmds, &cobra.Command{
			Use:   e.Name,
			Short: "deny access to " + env.Name,
			RunE: func(cmd *cobra.Command, args []string) error {
				lvl := out.GetLevel(env)

				// Build payload with action revoke
				payload, err := buildRemoteAccessPayload(cfg, "revoke")
				if err != nil {
					out.ErrorFHighlight("  %s %v", "✗", err)
					return nil
				}

				functionName := fmt.Sprintf("dis-%s-remote-allow", env.Name)
				// Get the AWS profile for this environment and command
				profile := cfg.GetProfileForCommand(env.Name, "remote.deny")

				out.Highlight(lvl, "invoking lambda %s for %s (profile: %s)", functionName, env.Name, profile)
				resp, err := aws.InvokeLambda(ctx, profile, functionName, payload)
				if err != nil {
					return fmt.Errorf("lambda invoke failed: %w", err)
				}
				var results []LambdaResult
				if err := json.Unmarshal([]byte(resp), &results); err != nil {
					cmd.Printf("%s\n", resp)
					return nil
				}
				user := ""
				if cfg.UserName != nil {
					user = *cfg.UserName
				}
				renderLambdaResults(lvl, env.Name, user, results)
				return nil
			},
		})
	}

	if len(envSubCmds) == 0 {
		out.Warn("Warning: No subcommands found for envs - missing envs in config?")
	}

	cmd.AddCommand(envSubCmds...)
	return cmd
}

// remoteLoginCommand creates the login subcommand for remote
func remoteLoginCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "login to AWS environment",
	}

	if len(cfg.Environments) > 0 {
		firstEnv := cfg.Environments[0]
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			lvl := out.GetLevel(firstEnv)
			loginCmd := "aws sso login --profile " + cfg.GetProfile(firstEnv.Name)
			out.Highlight(lvl, "logging in to %s using %s", firstEnv.Name, loginCmd)
			return cli.ExecCommand(ctx, loginCmd, ".")
		}
	} else {
		out.WarnFHighlight("Warning: No environments found in config - `dp remote login` will not work")
	}

	return cmd
}

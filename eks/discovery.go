package eks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ONSdigital/dp-cli/aws"
	sdkaws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

const (
	// DefaultTunnelBoxRoleTag is the default tag value used to discover the Session Tunnel Box
	DefaultTunnelBoxRoleTag = "session-tunnel-box"
	// DefaultClusterAccessTag is the default tag key used to identify clusters available for tunnel access
	DefaultClusterAccessTag = "ssm-tunnel-access"
)

// DiscoveryTags holds the resolved tag values for discovery
type DiscoveryTags struct {
	TunnelBoxRoleTag string
	ClusterAccessTag string
}

// NewDiscoveryTags returns tags with config overrides applied, falling back to defaults
func NewDiscoveryTags(tunnelBoxRoleTag, clusterAccessTag string) DiscoveryTags {
	tags := DiscoveryTags{
		TunnelBoxRoleTag: DefaultTunnelBoxRoleTag,
		ClusterAccessTag: DefaultClusterAccessTag,
	}
	if tunnelBoxRoleTag != "" {
		tags.TunnelBoxRoleTag = tunnelBoxRoleTag
	}
	if clusterAccessTag != "" {
		tags.ClusterAccessTag = clusterAccessTag
	}
	return tags
}

// ClusterInfo holds discovered EKS cluster information
type ClusterInfo struct {
	Name     string
	Endpoint string
}

// TunnelBoxInfo holds discovered tunnel box instance information
type TunnelBoxInfo struct {
	InstanceID string
	Name       string
}

// FindTunnelBox discovers the Secure Session Tunnel Box instance by tags.
func FindTunnelBox(ctx context.Context, profile string, tags DiscoveryTags) (*TunnelBoxInfo, error) {
	cfg, err := aws.GetAWSConfig(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := ec2.NewFromConfig(cfg)

	result, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name:   sdkaws.String("tag:Role"),
				Values: []string{tags.TunnelBoxRoleTag},
			},
			{
				Name:   sdkaws.String("instance-state-name"),
				Values: []string{"running"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe instances: %w", err)
	}

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			if instance.InstanceId == nil {
				continue
			}
			name := ""
			for _, tag := range instance.Tags {
				if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
					name = *tag.Value
				}
			}
			return &TunnelBoxInfo{
				InstanceID: *instance.InstanceId,
				Name:       name,
			}, nil
		}
	}

	return nil, fmt.Errorf("no running tunnel box found (looking for tag Role=%s)", tags.TunnelBoxRoleTag)
}

// FindClusters discovers EKS clusters available for tunnel access.
func FindClusters(ctx context.Context, profile string, tags DiscoveryTags) ([]ClusterInfo, error) {
	cfg, err := aws.GetAWSConfig(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := eks.NewFromConfig(cfg)

	listResult, err := client.ListClusters(ctx, &eks.ListClustersInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list EKS clusters: %w", err)
	}

	var clusters []ClusterInfo
	for _, clusterName := range listResult.Clusters {
		descResult, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{
			Name: sdkaws.String(clusterName),
		})
		if err != nil {
			continue
		}

		cluster := descResult.Cluster
		if cluster == nil || cluster.Tags == nil {
			continue
		}

		accessTag, hasAccess := cluster.Tags[tags.ClusterAccessTag]
		if !hasAccess || accessTag != "true" {
			continue
		}

		endpoint := ""
		if cluster.Endpoint != nil {
			endpoint = strings.TrimPrefix(*cluster.Endpoint, "https://")
		}

		clusters = append(clusters, ClusterInfo{
			Name:     clusterName,
			Endpoint: endpoint,
		})
	}

	if len(clusters) == 0 {
		return nil, fmt.Errorf("no EKS clusters found with tag %s=true", tags.ClusterAccessTag)
	}

	return clusters, nil
}

// ResolveEndpointIPv4 resolves an EKS endpoint to its IPv4 address via the tunnel box using SSM RunCommand.
// Uses the custom DIS-ResolveDNS document which restricts execution to dig lookups only.
// This is required for dualstack clusters to force A record resolution.
func ResolveEndpointIPv4(ctx context.Context, profile, tunnelBoxID, endpoint string) (string, error) {
	cfg, err := aws.GetAWSConfig(ctx, profile)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := ssm.NewFromConfig(cfg)

	sendResult, err := client.SendCommand(ctx, &ssm.SendCommandInput{
		InstanceIds:  []string{tunnelBoxID},
		DocumentName: sdkaws.String("DIS-ResolveDNS"),
		Parameters: map[string][]string{
			"hostname": {endpoint},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to send command to tunnel box: %w", err)
	}

	if sendResult.Command == nil || sendResult.Command.CommandId == nil {
		return "", fmt.Errorf("no command ID returned from SSM")
	}

	commandID := *sendResult.Command.CommandId

	var ipv4 string
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		if i == 0 {
			time.Sleep(3 * time.Second)
		} else {
			time.Sleep(2 * time.Second)
		}

		getResult, err := client.GetCommandInvocation(ctx, &ssm.GetCommandInvocationInput{
			CommandId:  sdkaws.String(commandID),
			InstanceId: sdkaws.String(tunnelBoxID),
		})
		if err != nil {
			continue
		}

		if getResult.Status == "Success" && getResult.StandardOutputContent != nil {
			ipv4 = strings.TrimSpace(*getResult.StandardOutputContent)
			break
		}
		if getResult.Status == "Failed" || getResult.Status == "Cancelled" {
			stderr := ""
			if getResult.StandardErrorContent != nil {
				stderr = *getResult.StandardErrorContent
			}
			return "", fmt.Errorf("command failed on tunnel box: %s", stderr)
		}
	}

	if ipv4 == "" {
		return "", fmt.Errorf("failed to resolve IPv4 for %s (timed out waiting for tunnel box response)", endpoint)
	}

	return ipv4, nil
}

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
	// BastionRoleTag is the tag value used to identify EKS bastion instances
	BastionRoleTag = "eks-session-tunnel"
	// ClusterAccessTag is the tag key used to identify clusters available for tunnel access
	ClusterAccessTag = "ssm-tunnel-access"
)

// ClusterInfo holds discovered EKS cluster information
type ClusterInfo struct {
	Name     string
	Endpoint string
}

// BastionInfo holds discovered bastion instance information
type BastionInfo struct {
	InstanceID string
	Name       string
}

// FindBastion discovers the EKS bastion instance by tags.
// Scoped to the AWS account via the profile.
func FindBastion(ctx context.Context, profile string) (*BastionInfo, error) {
	cfg, err := aws.GetAWSConfig(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := ec2.NewFromConfig(cfg)

	result, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name:   sdkaws.String("tag:Role"),
				Values: []string{BastionRoleTag},
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
			return &BastionInfo{
				InstanceID: *instance.InstanceId,
				Name:       name,
			}, nil
		}
	}

	return nil, fmt.Errorf("no running EKS bastion found (looking for tag Role=%s)", BastionRoleTag)
}

// FindClusters discovers EKS clusters available for tunnel access.
// Scoped to the AWS account via the profile — uses ssm-tunnel-access tag as opt-in.
func FindClusters(ctx context.Context, profile string) ([]ClusterInfo, error) {
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

		accessTag, hasAccess := cluster.Tags[ClusterAccessTag]
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
		return nil, fmt.Errorf("no EKS clusters found with tag %s=true", ClusterAccessTag)
	}

	return clusters, nil
}

// ResolveEndpointIPv4 resolves an EKS endpoint to its IPv4 address via the bastion using SSM RunCommand
func ResolveEndpointIPv4(ctx context.Context, profile, bastionID, endpoint string) (string, error) {
	cfg, err := aws.GetAWSConfig(ctx, profile)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := ssm.NewFromConfig(cfg)

	sendResult, err := client.SendCommand(ctx, &ssm.SendCommandInput{
		InstanceIds:  []string{bastionID},
		DocumentName: sdkaws.String("AWS-RunShellScript"),
		Parameters: map[string][]string{
			"commands": {fmt.Sprintf("dig +short A %s | head -1", endpoint)},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to send command to bastion: %w", err)
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
			InstanceId: sdkaws.String(bastionID),
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
			return "", fmt.Errorf("command failed on bastion: %s", stderr)
		}
	}

	if ipv4 == "" {
		return "", fmt.Errorf("failed to resolve IPv4 for %s (timed out waiting for bastion response)", endpoint)
	}

	return ipv4, nil
}

package framework

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GCPClient provides functions to interact with GCP using gcloud CLI.
type GCPClient struct {
	project string
	region  string
}

// LoadBalancerInfo represents information about a GCP load balancer.
type LoadBalancerInfo struct {
	Name        string `json:"name"`
	NetworkTier string `json:"networkTier"`
	IPAddress   string `json:"IPAddress"`
	Region      string `json:"region"`
}

// NewGCPClient creates a new GCP client instance.
func NewGCPClient(project, region string) *GCPClient {
	return &GCPClient{
		project: project,
		region:  region,
	}
}

// execGCloud executes a gcloud command and returns the output.
func (g *GCPClient) execGCloud(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "gcloud", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	klog.Infof("Executing: gcloud %s", strings.Join(args, " "))

	err := cmd.Run()
	output := strings.TrimSpace(stdout.String())

	if err != nil {
		return "", fmt.Errorf("gcloud command failed: %w, stderr: %s", err, stderr.String())
	}

	return output, nil
}

// GetLoadBalancerInfo retrieves information about a load balancer by IP address.
func (g *GCPClient) GetLoadBalancerInfo(ctx context.Context, ipAddress string) (*LoadBalancerInfo, error) {
	// Validate IP address is not empty
	if ipAddress == "" {
		return nil, fmt.Errorf("IP address cannot be empty")
	}

	// First, try to find the forwarding rule by IP address.
	args := []string{
		"compute", "forwarding-rules", "list",
		"--filter", fmt.Sprintf("IPAddress=%s", ipAddress),
		"--format", "json",
		"--project", g.project,
	}

	if g.region != "" {
		args = append(args, "--regions", g.region)
	}

	output, err := g.execGCloud(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list forwarding rules: %w", err)
	}

	var forwardingRules []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &forwardingRules); err != nil {
		return nil, fmt.Errorf("failed to parse forwarding rules JSON: %w", err)
	}

	klog.Infof("Found %d forwarding rules for IP %s", len(forwardingRules), ipAddress)

	if len(forwardingRules) == 0 {
		return nil, fmt.Errorf("no forwarding rule found for IP address %s in project %s region %s", ipAddress, g.project, g.region)
	}

	rule := forwardingRules[0]

	// Extract network tier information.
	networkTier := "Standard" // default
	if tier, ok := rule["networkTier"].(string); ok {
		networkTier = strings.ToUpper(tier)
	}

	// Extract name.
	name := ""
	if n, ok := rule["name"].(string); ok {
		name = n
	}

	// Extract region.
	region := g.region

	if r, ok := rule["region"].(string); ok {
		// Extract region name from full URL.
		parts := strings.Split(r, "/")
		if len(parts) > 0 {
			region = parts[len(parts)-1]
		}
	}

	return &LoadBalancerInfo{
		Name:        name,
		NetworkTier: networkTier,
		IPAddress:   ipAddress,
		Region:      region,
	}, nil
}

// isGCloudAvailable checks if gcloud CLI is available in the system.
func (g *GCPClient) isGCloudAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "gcloud", "version")
	err := cmd.Run()

	return err == nil
}

// WaitForLoadBalancerNetworkTier waits for a load balancer to have the expected network tier.
func (g *GCPClient) WaitForLoadBalancerNetworkTier(ctx context.Context, ipAddress, expectedTier string, timeout time.Duration) error {
	// Check if gcloud is available first
	if !g.isGCloudAvailable(ctx) {
		return fmt.Errorf("gcloud CLI is not available in the system")
	}

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check if context is cancelled.
		if ctx.Err() != nil {
			return ctx.Err()
		}

		info, err := g.GetLoadBalancerInfo(ctx, ipAddress)
		if err != nil {
			klog.Infof("Failed to get load balancer info: %v, retrying...", err)
			time.Sleep(10 * time.Second)

			continue
		}

		if strings.EqualFold(info.NetworkTier, expectedTier) {
			klog.Infof("Load balancer %s has expected network tier: %s", ipAddress, info.NetworkTier)
			return nil
		}

		klog.Infof("Load balancer %s has network tier: %s, expected: %s, waiting...",
			ipAddress, info.NetworkTier, expectedTier)
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("timeout waiting for load balancer %s to have network tier %s", ipAddress, expectedTier)
}

// VerifyLoadBalancerCreation verifies that a load balancer was created successfully.
func (g *GCPClient) VerifyLoadBalancerCreation(ctx context.Context, ipAddress string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check if context is cancelled.
		if ctx.Err() != nil {
			return ctx.Err()
		}

		_, err := g.GetLoadBalancerInfo(ctx, ipAddress)
		if err == nil {
			klog.Infof("Load balancer %s created successfully", ipAddress)
			return nil
		}

		klog.Infof("Load balancer %s not found yet, retrying...", ipAddress)
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("timeout waiting for load balancer %s to be created", ipAddress)
}

// GetGCPCredentialsFromInfrastructure retrieves GCP project and region from the infrastructure object.
func GetGCPCredentialsFromInfrastructure(ctx context.Context, cl client.Client) (string, string, error) {
	infra := &configv1.Infrastructure{}

	err := cl.Get(ctx, client.ObjectKey{
		Name: "cluster",
	}, infra)
	if err != nil {
		return "", "", fmt.Errorf("failed to get infrastructure object: %w", err)
	}

	if infra.Status.PlatformStatus == nil || infra.Status.PlatformStatus.Type != configv1.GCPPlatformType {
		return "", "", fmt.Errorf("infrastructure is not GCP platform")
	}

	gcpStatus := infra.Status.PlatformStatus.GCP
	if gcpStatus == nil {
		return "", "", fmt.Errorf("GCP platform status is nil")
	}

	if gcpStatus.ProjectID == "" {
		return "", "", fmt.Errorf("GCP project ID is empty")
	}

	if gcpStatus.Region == "" {
		return "", "", fmt.Errorf("GCP region is empty")
	}

	return gcpStatus.ProjectID, gcpStatus.Region, nil
}

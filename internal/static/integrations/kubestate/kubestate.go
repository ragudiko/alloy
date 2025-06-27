package kubestate

import (
	"context"
	"fmt"
	"os"

	"github.com/go-kit/log"

	"github.com/grafana/alloy/internal/static/integrations"
	"github.com/grafana/ckit"

	//ksm

	// ksmcollectors "k8s.io/kube-state-metrics/v2/pkg/collectors"
	// "k8s.io/kube-state-metrics/v2/pkg/factory"

	// "k8s.io/kube-state-metrics/pkg/options"

	"strings"

	"k8s.io/kube-state-metrics/v2/pkg/app"
	ksmconfig "k8s.io/kube-state-metrics/v2/pkg/options"
)

// NewIntegration creates a new kubestate integration
func (c *Config) NewIntegration(logger log.Logger) (integrations.Integration, error) {
	fmt.Printf("NewIntegration method inside integration\n")
	return New(logger, c)
}

func toResourceSet(resources []string) ksmconfig.ResourceSet {
	rs := make(ksmconfig.ResourceSet)
	for _, r := range resources {
		rs[r] = struct{}{}
	}
	return rs
}

func New(logger log.Logger, cfg *Config) (integrations.Integration, error) {

	fmt.Printf("New method inside integration\n")

	ksmOpts := &ksmconfig.Options{
		// MetricAllowlist: cfg.MetricAllowlist,
		Namespaces:    cfg.Namespaces,
		Resources:     toResourceSet(cfg.Resources),
		Port:          cfg.Port,
		TelemetryPort: cfg.TelemetryPort,
		Node:          ksmconfig.NodeType(cfg.NodeName),
	}

	fmt.Println("ksmOpts port Before\n", ksmOpts.Port)
	ksmOpts.Port = 37425
	ksmOpts.Host = "::"
	ksmOpts.TelemetryPort = 9090
	ksmOpts.TotalShards = 1
	fmt.Println("ksmOpts port\n", ksmOpts.Port, " \nksmOpts.TelemetryPort\n", ksmOpts.TelemetryPort)
	fmt.Println("ksmOpts Apiserver, Host\n", ksmOpts.Apiserver, ksmOpts.Host)
	fmt.Println("ksmOpts.Node.String(), ksmOpts.Pod\n", ksmOpts.Node.String(), ksmOpts.Pod)
	fmt.Println("ksmOpts Config\n", ksmOpts.Config)

	ckitConfig := ckit.Config{
		Name:  cfg.NodeName, // we can use this instead of os.Getenv("NODE_NAME")
		Label: cfg.ClusterName,
		// Sharder:       shard.Ring(512),
		// EnableTLS: opts.EnableTLS,
	}

	fmt.Println("cluster.Cluster", ckitConfig.Name)
	// ckitConfig.Name = os.Getenv("NODE_NAME")
	fmt.Println("os.Getenv(\"NODE_NAME\")", ckitConfig.Name)
	fmt.Println("ckitConfig Node:, ClusterName:", ckitConfig.Name, ckitConfig.Label)

	ckitConfig.Name, _ = os.Hostname()
	fmt.Println("os.Hostname()", ckitConfig.Name)
	ckitConfig.Name = GetNodeName()
	fmt.Println("GetNodeName()", GetNodeName())

	return integrations.NewCollectorIntegration(
		cfg.Name(),
		integrations.WithRunner(func(ctx context.Context) error {
			if strings.Contains(ckitConfig.Name, "k3d-three-node-cluster-agent-2") ||
				strings.Contains(ckitConfig.Name, "k3d-three-node-cluster-agent-0") {
				app.RunKubeStateMetrics(ctx, ksmOpts)
			}
			return nil
		}),
	), nil
}

func GetNodeName() string {
	if name := os.Getenv("NODE_NAME"); name != "" {
		return name
	}
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

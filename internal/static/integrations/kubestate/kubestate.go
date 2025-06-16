package kubestate

import (
	"context"
	"fmt"

	"github.com/go-kit/log"

	"github.com/grafana/alloy/internal/static/integrations"

	//ksm

	// ksmcollectors "k8s.io/kube-state-metrics/v2/pkg/collectors"
	// "k8s.io/kube-state-metrics/v2/pkg/factory"

	// "k8s.io/kube-state-metrics/pkg/options"

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
	}
	fmt.Println("ksmOpts port Before\n", ksmOpts.Port)
	ksmOpts.Port = 37425
	ksmOpts.Host = "::"
	ksmOpts.TelemetryPort = 9090
	ksmOpts.TotalShards = 1
	fmt.Println("ksmOpts port\n", ksmOpts.Port, " \nksmOpts.TelemetryPort\n", ksmOpts.TelemetryPort)

	return integrations.NewCollectorIntegration(
		cfg.Name(),
		integrations.WithRunner(func(ctx context.Context) error {
			app.RunKubeStateMetrics(ctx, ksmOpts)
			return nil
		}),
	), nil
}

package kubestate

import (
	"context"
	"fmt"

	"github.com/go-kit/log"

	// "github.com/google/cadvisor/cache/memory"
	// "github.com/google/cadvisor/container"
	// v2 "github.com/google/cadvisor/info/v2"
	// "github.com/google/cadvisor/manager"
	// "github.com/google/cadvisor/metrics"
	// "github.com/google/cadvisor/storage"
	// "github.com/google/cadvisor/utils/sysfs"
	// "k8s.io/utils/clock"

	"github.com/grafana/alloy/internal/static/integrations"
	// Register container providers
	// "github.com/google/cadvisor/container/containerd"
	// "github.com/google/cadvisor/container/crio"
	// "github.com/google/cadvisor/container/docker"
	// "github.com/google/cadvisor/container/raw"
	// "github.com/google/cadvisor/container/systemd"

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

// New creates a new kubestate integration

// func New(logger log.Logger, c *Config) (integrations.Integration, error) {
// 	fmt.Printf("New method inside integration\n")
// 	// Set up Kubernetes client
// 	// restCfg, err := k8sConfig.InClusterConfig()
// 	// if err != nil {
// 	// 	return nil, err
// 	// }
// 	// kubeClient, err := kubernetes.NewForConfig(restCfg)
// 	// if err != nil {
// 	// 	return nil, err
// 	// }

// 	// Create kube-state-metrics config

// 	ksmCfg := ksmconfig.NewOptions()
// 	includeNamespaces := []string{"default", "kube-system"}
// 	ksmCfg.Namespaces = ksmconfig.NamespaceList{includeNamespaces[0], includeNamespaces[1]}

// 	// ksmCfg.Resources = []string{"pods", "deployments"}
// 	ksmCfg.Resources = ksmconfig.DefaultResources
// 	// Get enabled metric generators

// 	ksmCfg.MetricAllowlist = ksmconfig.MetricSet{
// 		"kube_pod_info":                 {},
// 		"kube_deployment_spec_replicas": {},
// 	}
// 	ksmCfg.MetricDenylist = ksmconfig.MetricSet{
// 		"kube_pod_status_phase": {},
// 	}

// 	// Build collectors from generators
// 	pcollectors := []prometheus.Collector{}

// 	// ksmMetricsRegistry := prometheus.NewRegistry()
// 	// ksmMetricsRegistry.MustRegister(
// 	// 	collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
// 	// 	collectors.NewGoCollector(),
// 	// )

// 	// Create and return an integration
// 	return integrations.NewCollectorIntegration(
// 		c.Name(),
// 		integrations.WithCollectors(pcollectors...),
// 	), nil
// }

func New(logger log.Logger, cfg *Config) (integrations.Integration, error) {

	ksmOpts := &ksmconfig.Options{
		MetricAllowlist: cfg.MetricAllowlist,
		// Namespaces:      cfg.Namespaces,
		Resources: cfg.Resources,
		Port:      8080,
	}

	return integrations.NewCollectorIntegration(
		cfg.Name(),
		integrations.WithRunner(func(ctx context.Context) error {
			app.RunKubeStateMetrics(ctx, ksmOpts)
			return nil
		}),
	), nil
}

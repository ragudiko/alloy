package kubestate

import (
	"fmt"

	"github.com/grafana/alloy/internal/component"
	"github.com/grafana/alloy/internal/component/common/kubernetes"
	"github.com/grafana/alloy/internal/component/prometheus/exporter"
	"github.com/grafana/alloy/internal/featuregate"
	"github.com/grafana/alloy/internal/service/cluster"
	"github.com/grafana/alloy/internal/static/integrations"

	// "k8s.io/client-go/tools/cache"

	"github.com/grafana/alloy/internal/static/integrations/kubestate"
)

func init() {
	fmt.Printf("***************init method inside exporter=============================\n")
	component.Register(component.Registration{
		Name:      "prometheus.exporter.kubestate",
		Stability: featuregate.StabilityGenerallyAvailable,
		Args:      Arguments{},
		Exports:   exporter.Exports{},

		Build: exporter.New(createExporter, "kubestate"),
	})
}

func createExporter(opts component.Options, args component.Arguments, defaultInstanceKey string) (integrations.Integration, string, error) {
	fmt.Printf("createExporter method \n")
	a := args.(Arguments)
	return integrations.NewIntegrationWithInstanceKey(opts.Logger, a.Convert(), defaultInstanceKey)
}

// TODO: replace cadvisor with kubestate
func (a *Arguments) Convert() *kubestate.Config {
	fmt.Printf("Convert method \n")

	cfg := &kubestate.Config{
		Client:          a.Client,
		PollFrequency:   a.PollFrequency,
		PollTimeout:     a.PollTimeout,
		Clustering:      a.Clustering,
		HTTPListenPort:  a.HTTPListenPort,
		Resources:       a.Resources,
		MetricAllowlist: a.MetricAllowlist,
		Namespaces:      a.Namespaces,
		KubeConfig:      a.KubeConfig,
	}

	return cfg
}

// DefaultArguments holds the default settings for the prometheus.exporter.kubestate component.
var DefaultArguments = Arguments{
	PollFrequency: "1m",
	PollTimeout:   "15s",
}

// Arguments configures the prometheus.exporter.kubestate component.
type Arguments struct {
	// Client settings to connect to Kubernetes.
	Client kubernetes.ClientArguments `alloy:"client,block,optional"`

	// How often to poll the Kubernetes API for metrics.
	PollFrequency string `alloy:"poll_frequency,attr,optional"`

	// Timeout when polling the Kubernetes API.
	PollTimeout string `alloy:"poll_timeout,attr,optional"`

	// List of Kubernetes resources to collect metrics for.
	// If empty, all supported resources will be collected.
	Resources []string `alloy:"resources,attr,optional"`

	// Clustering configuration for leader election
	Clustering cluster.ComponentBlock `alloy:"clustering,block,optional"`

	HTTPListenPort int `alloy:"http_listen_port,attr,optional"`

	Port int `alloy:"port,attr"`

	MetricAllowlist []string `alloy:"metric_allowlist,attr,optional"`
	Namespaces      []string `alloy:"namespaces,attr,optional"`

	KubeConfig string `alloy:"kube_config,attr,optional"`

	TelemetryPort int `alloy:"telemetry_port,attr,optional"`
}

func (a *Arguments) SetToDefault() {
	*a = DefaultArguments
}

// func toResourceSet(resources []string) options.ResourceSet {
// 	rs := make(options.ResourceSet)
// 	for _, r := range resources {
// 		rs[r] = struct{}{}
// 	}
// 	return rs
// }

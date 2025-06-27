package kubestate

import (
	"fmt"

	"github.com/go-kit/log"
	"github.com/grafana/alloy/internal/component/common/kubernetes"
	"github.com/grafana/alloy/internal/service/cluster"
	"github.com/grafana/alloy/internal/static/integrations"
	integrations_v2 "github.com/grafana/alloy/internal/static/integrations/v2"
	"github.com/grafana/alloy/internal/static/integrations/v2/metricsutils"

	//clustering
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

const name = "kubestate"

// DefaultConfig holds the default settings for the kubestate integration
var DefaultConfig = Config{}

// Config controls kubestate
type Config struct {
	// Client settings to connect to Kubernetes.
	Client kubernetes.ClientArguments `alloy:"client,block,optional"`

	// How often to poll the Kubernetes API for metrics.
	PollFrequency string `alloy:"poll_frequency,attr,optional"`

	// Timeout when polling the Kubernetes API.
	PollTimeout string `alloy:"poll_timeout,attr,optional"`

	// List of Kubernetes resources to collect metrics for.
	// If empty, all supported resources will be collected.
	// Resources []string `alloy:"resources,attr,optional"`

	// Clustering configuration for leader election
	Clustering cluster.ComponentBlock `alloy:"clustering,block,optional"`

	HTTPListenPort int `alloy:"http_listen_port,attr,optional"`

	Port int `alloy:"ksm_port,attr,optional"`

	// Hold on to the logger passed to config.NewIntegration, to be passed to klog, as yet another unsafe global that needs to be set.
	logger log.Logger `alloy:"logger,attr,optional"` // logger is only used on linux

	MetricAllowlist []string `alloy:"metric_allowlist,attr,optional"`
	Namespaces      []string `alloy:"namespaces,attr,optional"`
	Resources       []string `alloy:"resources,attr,optional"`

	KubeConfig string `alloy:"kube_config,attr,optional"`

	TelemetryPort int `alloy:"telemetry_port,attr,optional"`

	//========================= clustering starts ==========================

	Log     log.Logger            // Where to send logs to.
	Metrics prometheus.Registerer // Where to send metrics to.
	Tracer  trace.TracerProvider  // Where to send traces.

	// EnableClustering toggles clustering as a whole. When EnableClustering is
	// false, the instance of Alloy acts as a single-node cluster, and it is not
	// possible for other nodes to join the cluster.
	EnableClustering bool

	NodeName         string // Name to use for this node in the cluster.
	AdvertiseAddress string // Address to advertise to other nodes in the cluster.
	EnableTLS        bool   // Specifies whether TLS should be used for communication between peers.
	TLSCAPath        string // Path to the CA file.
	TLSCertPath      string // Path to the certificate file.
	TLSKeyPath       string // Path to the key file.
	TLSServerName    string // Server name to use for TLS communication.
	// RejoinInterval         time.Duration // How frequently to rejoin the cluster to address split brain issues.
	ClusterMaxJoinPeers int    // Number of initial peers to join from the discovered set.
	ClusterName         string // Name to prevent nodes without this identifier from joining the cluster.
	MinimumClusterSize  int    // Minimum cluster size before admitting traffic to components that use clustering.
	// MinimumSizeWaitTimeout time.Duration // Maximum duration to wait for minimum cluster size before proceeding; 0 means no timeout.

	// Function to discover peers to join. If this function is nil or returns an
	// empty slice, no peers will be joined.
	// DiscoverPeers discovery.DiscoverFn

	// ========================= clustering =========================
}

// UnmarshalYAML implements yaml.Unmarshaler for Config
// func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
// 	*c = DefaultConfig

// 	type plain Config
// 	err := unmarshal((*plain)(c))
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// Name returns the name of the integration that this config represents.
func (c *Config) Name() string {
	return name
}

// InstanceKey returns the agentKey
func (c *Config) InstanceKey(agentKey string) (string, error) {
	return agentKey, nil
}

func init() {
	fmt.Printf("init method inside integration\n")
	integrations.RegisterIntegration(&Config{})
	integrations_v2.RegisterLegacy(&Config{}, integrations_v2.TypeSingleton, metricsutils.Shim)
}

package kubestate

import (
	"fmt"
	"log"
	"sync/atomic"

	"github.com/grafana/alloy/internal/component"
	"github.com/grafana/alloy/internal/component/common/kubernetes"
	"github.com/grafana/alloy/internal/component/prometheus/exporter"
	"github.com/grafana/alloy/internal/featuregate"
	"github.com/grafana/alloy/internal/service/cluster"
	clusterpkg "github.com/grafana/alloy/internal/service/cluster"
	"github.com/grafana/alloy/internal/static/integrations"
	"github.com/grafana/alloy/internal/static/integrations/kubestate"
	"github.com/grafana/ckit/shard"
	// "k8s.io/client-go/tools/cache"
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

	leaderImpl := &componentLeadership{}

	_, err := leaderImpl.update()
	if err != nil {
		fmt.Println("msg", "checking leadership during starting failed, will retry", "err", err)
	}
	leaderImpl.isLeader()
	fmt.Println(" \nleaderImpl.isLeader() ======================", leaderImpl.isLeader())
}

// func startup() {
// 	panic("unimplemented")
// }

//	func (c *Component) startup() {
//		if !c.leader.isLeader() {
//			fmt.Println("msg", "skipping startup because we are not the leader")
//		}
//	}
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
		Cluster:         a.Cluster,
	}

	return cfg
}

type Component struct {
	log  log.Logger
	opts component.Options
	args Arguments

	leader leadership
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

	Cluster cluster.Cluster `alloy:"cluster,attr,optional"`
}

func (a *Arguments) SetToDefault() {
	*a = DefaultArguments
}

// leadership encapsulates the logic for checking if this instance of the Component
// is the leader among all instances to avoid conflicting updates of the Mimir API.
type leadership interface {
	// update checks if this component instance is still the leader, stores the result,
	// and returns true if the leadership status has changed since the last time update
	// was called.
	update() (bool, error)

	// isLeader returns true if this component instance is the leader, false otherwise.
	isLeader() bool
}

// componentLeadership implements leadership based on checking ownership of a specific
// key using a cluster.Cluster service.
type componentLeadership struct {
	id     string
	logger log.Logger
	cl     clusterpkg.Cluster
	leader atomic.Bool
}

func newComponentLeadership(id string, logger log.Logger, cl clusterpkg.Cluster) *componentLeadership {
	return &componentLeadership{
		id:     id,
		logger: logger,
		cl:     cl,
	}
}

func (l *componentLeadership) update() (bool, error) {
	// NOTE: since this is leader election, it is okay to NOT check if cluster is ready.
	peers, err := l.cl.Lookup(shard.StringKey(l.id), 1, shard.OpReadWrite)
	if err != nil {
		return false, fmt.Errorf("unable to determine leader for %s: %w", l.id, err)
	}

	if len(peers) != 1 {
		return false, fmt.Errorf("unexpected peers from leadership check: %+v", peers)
	}

	isLeader := peers[0].Self
	fmt.Println("msg", "checked leadership of component", "is_leader", isLeader)
	return l.leader.Swap(isLeader) != isLeader, nil
}

func (l *componentLeadership) isLeader() bool {
	return l.leader.Load()
}

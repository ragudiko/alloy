package kubestate

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/alloy/internal/component"
	"github.com/grafana/alloy/internal/component/common/kubernetes"
	"github.com/grafana/alloy/internal/component/prometheus/exporter"
	"github.com/grafana/alloy/internal/featuregate"
	"github.com/grafana/alloy/internal/service/cluster"
	"github.com/grafana/alloy/internal/static/integrations"
	"github.com/grafana/ckit/shard"

	// "k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/pkg/builder"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"
	"k8s.io/kube-state-metrics/v2/pkg/options"

	"github.com/grafana/alloy/internal/static/integrations/kubestate"
	kube "k8s.io/client-go/kubernetes"
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
	// if len(a.PollFrequency) == 0 {
	// 	a.PollFrequency = string{""}
	// }
	// if len(a.PollTimeout) == 0 {
	// 	a.PollTimeout = string{""}
	// }

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
	Resources options.ResourceSet `alloy:"resources,attr,optional"`

	// Clustering configuration for leader election
	Clustering cluster.ComponentBlock `alloy:"clustering,block,optional"`

	HTTPListenPort int `river:"http_listen_port,attr"`

	MetricAllowlist options.MetricSet     `alloy:"metric_allowlist,attr,optional"`
	Namespaces      options.NamespaceList `alloy:"namespaces,attr,optional"`

	KubeConfig string `alloy:"kube_config,attr,optional"`
}

func (a *Arguments) SetToDefault() {
	*a = DefaultArguments
}

// Component implements the prometheus.exporter.kubestate component.
type Component struct {
	log        log.Logger
	opts       component.Options
	args       Arguments
	builder    *builder.Builder
	stores     []*metricsstore.MetricsStore
	httpServer *http.Server
	cluster    cluster.Cluster
	leader     *componentLeadership
}

// componentLeadership implements leader election using cluster.Cluster
type componentLeadership struct {
	id      string
	cluster cluster.Cluster
	leader  bool
}

func newComponentLeadership(id string, cluster cluster.Cluster) *componentLeadership {
	return &componentLeadership{
		id:      id,
		cluster: cluster,
	}
}

func (l *componentLeadership) update() (bool, error) {
	peers, err := l.cluster.Lookup(shard.StringKey(l.id), 1, shard.OpReadWrite)
	if err != nil {
		return false, fmt.Errorf("unable to determine leader for %s: %w", l.id, err)
	}

	if len(peers) != 1 {
		return false, fmt.Errorf("unexpected peers from leadership check: %+v", peers)
	}

	isLeader := peers[0].Self
	changed := l.leader != isLeader
	l.leader = isLeader
	return changed, nil
}

func (l *componentLeadership) isLeader() bool {
	return l.leader
}

// New creates a new prometheus.exporter.kubestate component.
func New(o component.Options, args Arguments) (*Component, error) {
	fmt.Printf("New method inside exporter \n")
	clusterSvc, err := o.GetServiceData(cluster.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("getting cluster service failed: %w", err)
	}

	return &Component{
		opts:    o,
		args:    args,
		cluster: clusterSvc.(cluster.Cluster),
		leader:  newComponentLeadership(o.ID, clusterSvc.(cluster.Cluster)),
	}, nil
}

func toResourceSet(resources []string) options.ResourceSet {
	rs := make(options.ResourceSet)
	for _, r := range resources {
		rs[r] = struct{}{}
	}
	return rs
}

// Run starts the prometheus.exporter.kubestate component.
func (c *Component) Run(ctx context.Context) error { // The `Run` method inside the `Component` struct of the
	// `prometheus.exporter.kubestate` component is responsible for starting the
	// component. Here is a breakdown of what the `Run` method is doing:

	fmt.Printf("Run method inside exporter \n")
	// Create Kubernetes client config
	config, err := c.args.Client.BuildRESTConfig(c.log)
	if err != nil {
		return fmt.Errorf("building Kubernetes client config: %w", err)
	}

	kubeClient, err := kube.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubeClient: %w", err)
	}

	// discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	// if err != nil {
	// 	return fmt.Errorf("failed to create discoveryClient: %w", err)
	// }

	// Create kube-state-metrics builder
	ksmOptions := options.NewOptions()
	if len(c.args.Resources) > 0 {
		ksmOptions.Resources = toResourceSet(c.args.Resources.AsSlice())
	}

	c.builder = builder.NewBuilder()
	// c.builder.WithKubeConfig(config)
	c.builder.WithKubeClient(kubeClient)
	c.builder.WithNamespaces(options.DefaultNamespaces)
	c.builder.WithSharding(0, 1)
	c.builder.WithContext(ctx)

	// Build stores
	// c.stores = c.builder.Build()
	writers := c.builder.Build()

	c.stores = make([]*metricsstore.MetricsStore, 0, len(writers))
	if err != nil {
		return fmt.Errorf("building kube-state-metrics stores: %w", err)
	}

	// Start stores
	for _, store := range c.stores {
		// store.Start()
		store.List()
	}

	// Create HTTP server for metrics endpoint
	mux := http.NewServeMux()
	// mux.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	// Only serve metrics if we are the leader
	// 	if !c.leader.isLeader() {
	// 		http.Error(w, "Not the leader", http.StatusServiceUnavailable)
	// 		return
	// 	}
	// 	for _, store := range c.stores {
	// 		store.WriteAll(w)
	// 	}
	// }))

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		for _, writer := range writers {
			writer.WriteAll(w)
		}
	})

	c.httpServer = &http.Server{
		// Addr:    fmt.Sprintf(":%d", c.opts.HTTPListenPort),
		Addr:    fmt.Sprintf(":%d", c.args.HTTPListenPort),
		Handler: mux,
	}

	// Start HTTP server
	go func() {
		if err := c.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// c.opts.Logger.Error("HTTP server error", "error", err)
			c.opts.Logger.Log("HTTP server error", "error", err)
		}
	}()

	// Start leader election loop
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				changed, err := c.leader.update()
				if err != nil {
					// c.opts.Logger.Error("Failed to check leadership", "error", err)
					c.opts.Logger.Log("Failed to check leadership", "error", err)
					continue
				}
				if changed {
					c.opts.Logger.Log("Leadership status changed", "is_leader", c.leader.isLeader())
				}
			}
		}
	}()

	<-ctx.Done()
	return c.httpServer.Shutdown(context.Background())
}

// Update updates the prometheus.exporter.kubestate component.
func (c *Component) Update(args Arguments) error {
	c.args = args
	return nil
}

// Name returns the name of the component.
func (c *Component) Name() string {
	return "prometheus.exporter.kubestate"
}

// Exports returns the values exported by the component.
func (c *Component) Exports() map[string]interface{} {
	fmt.Printf("Exports method inside exporter\n")
	// return exporter.DefaultExports(c.opts)
	return c.Exports()
}

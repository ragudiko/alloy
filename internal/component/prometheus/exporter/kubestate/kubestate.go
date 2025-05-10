package kubestate

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/alloy/internal/component"
	"github.com/grafana/alloy/internal/component/common/kubernetes"
	"github.com/grafana/alloy/internal/component/prometheus/exporter"
	"github.com/grafana/alloy/internal/service/cluster"
	"github.com/grafana/ckit/shard"
	"k8s.io/kube-state-metrics/v2/pkg/builder"
	// "k8s.io/kube-state-metrics/v2/pkg/metricsstore"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

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
}

func (a *Arguments) SetToDefault() {
	*a = DefaultArguments
}

// Component implements the prometheus.exporter.kubestate component.
type Component struct {
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

// Run starts the prometheus.exporter.kubestate component.
func (c *Component) Run(ctx context.Context) error {
	// Create Kubernetes client config
	config, err := c.args.Client.BuildRESTConfig()
	if err != nil {
		return fmt.Errorf("building Kubernetes client config: %w", err)
	}

	// Create kube-state-metrics builder
	ksmOptions := options.NewOptions()
	if len(c.args.Resources) > 0 {
		ksmOptions.Resources = c.args.Resources
	}

	c.builder = builder.NewBuilder()
	c.builder.WithKubeConfig(config)
	c.builder.WithNamespaces(options.DefaultNamespaces)
	c.builder.WithSharding(0, 1)
	c.builder.WithContext(ctx)

	// Build stores
	c.stores, err = c.builder.Build()
	if err != nil {
		return fmt.Errorf("building kube-state-metrics stores: %w", err)
	}

	// Start stores
	for _, store := range c.stores {
		store.Start()
	}

	// Create HTTP server for metrics endpoint
	mux := http.NewServeMux()
	mux.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only serve metrics if we are the leader
		if !c.leader.isLeader() {
			http.Error(w, "Not the leader", http.StatusServiceUnavailable)
			return
		}
		for _, store := range c.stores {
			store.WriteAll(w)
		}
	}))

	c.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", c.opts.HTTPListenPort),
		Handler: mux,
	}

	// Start HTTP server
	go func() {
		if err := c.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			c.opts.Logger.Error("HTTP server error", "error", err)
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
					c.opts.Logger.Error("Failed to check leadership", "error", err)
					continue
				}
				if changed {
					c.opts.Logger.Info("Leadership status changed", "is_leader", c.leader.isLeader())
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
	return exporter.DefaultExports(c.opts)
}
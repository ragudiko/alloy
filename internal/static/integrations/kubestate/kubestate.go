package kubestate

import (
	"context"
	"fmt"
	"os"

	"github.com/grafana/alloy/internal/static/integrations"
	"github.com/grafana/ckit"

	//ksm

	// ksmcollectors "k8s.io/kube-state-metrics/v2/pkg/collectors"
	// "k8s.io/kube-state-metrics/v2/pkg/factory"

	// "k8s.io/kube-state-metrics/pkg/options"

	"strings"

	"regexp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kube-state-metrics/v2/pkg/app"
	ksmconfig "k8s.io/kube-state-metrics/v2/pkg/options"

	"github.com/go-kit/log"
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
	fmt.Println("kubestate.go  ksmOpts port\n", ksmOpts.Port, " \nksmOpts.TelemetryPort\n", ksmOpts.TelemetryPort)
	fmt.Println("kubestate.go  ksmOpts Apiserver, Host\n", ksmOpts.Apiserver, ksmOpts.Host)
	fmt.Println("kubestate.go  ksmOpts.Node.String(), ksmOpts.Pod\n", ksmOpts.Node.String(), ksmOpts.Pod)
	fmt.Println("kubestate.go  ksmOpts Config\n", ksmOpts.Config)

	ckitConfig := &ckit.Config{
		Name:  cfg.NodeName, // we can use this instead of os.Getenv("NODE_NAME")
		Label: cfg.ClusterName,
		// Sharder:       shard.Ring(512),
		// EnableTLS: opts.EnableTLS,
	}

	ckitConfig.Name, _ = os.Hostname()
	fmt.Println("==================kubestate.go os.Hostname()", ckitConfig.Name)
	ckitConfig.Name = GetNodeName()
	fmt.Println("==================kubestate.go GetNodeName()", GetNodeName())

	leader := os.Getenv("NODE_REGEX")
	fmt.Println("==================kubestate.go os.Getenv(\"NODE_REGEX\")", leader)
	// err, isLeader := getLeader(Config.cluster)
	// var l leadership = &componentLeadership{}
	// if l.isLeader() {
	// 	fmt.Println("This node is the leader")
	// }

	return integrations.NewCollectorIntegration(
		cfg.Name(),
		integrations.WithRunner(func(ctx context.Context) error {
			//requiredNode := GetNodeNameRegex(ckitConfig.Name, pattern)
			nodeName, err := getNodeNameFromPod(leader)
			if err != nil {
				fmt.Errorf("failed to get node name %w", err)
			}
			fmt.Println("msg=====================\n", "nodeName", nodeName, "\n=====================")
			if strings.Contains(ckitConfig.Name, nodeName) {
				app.RunKubeStateMetrics(ctx, ksmOpts)
			}
			// peers := cluster.Cluster.Peers()
			// isLeader := peers[0].Self
			// fmt.Println("==================kubestate.go isLeader", isLeader)
			return nil
		}),
	), nil
}

// func getLeader2(l leadership) {
// 	l.update()
// 	if l.isLeader() {

// 	}
// }

// func dummy(){
// 	var leadership componentLeadership
// 	l:=leadership.isLeader()
// 	fmt.Println(l)
// }

// func getNodefromPod() string {

// 	var pod corev1.Pod
// 	nodeName := pod.Spec.NodeName
// 	podIP := pod.Status.PodIP

// 	podname := &corev1.Pod

// 	fmt.Println("msg", "nodeName", nodeName, "\npodIP ", podIP)
// 	return nodeName
// }

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

func GetNodeNameRegex(nodeName, regexPattern string) string {
	matched, err := regexp.MatchString(regexPattern, nodeName)
	if err != nil {
		// Optionally log the regex error
		return "node names not matching the regex - NODE_REGEX"
	}
	if matched {
		return nodeName
	}
	return ""
}

// func getLeader(c cluster.Cluster) (bool, error) {
// 	// NOTE: since this is leader election, it is okay to NOT check if cluster is ready.
// 	peers := c.Peers()
// 	// if err != nil {
// 	// 	return false, fmt.Errorf("unable to determine leader for %s: %w", l.id, err)
// 	// }

// 	if len(peers) != 1 {
// 		return false, fmt.Errorf("unexpected peers from leadership check: %+v", peers)
// 	}

// 	isLeader := peers[0].Self
// 	fmt.Println("peers[0].Name", peers[0].Name)

// 	fmt.Println("msg", "checked leadership of component", "is_leader", isLeader)
// 	return isLeader, nil
// }

func getNodeNameFromPod(leaderpod string) (string, error) {
	// Get pod name (hostname by default)
	podName := leaderpod
	if podName == "" {
		return "", fmt.Errorf("HOSTNAME env var is not set")
	}

	// Get namespace (usually injected via Downward API or file)
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		// Fallback to default method: read from service account namespace file
		data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			return "", fmt.Errorf("unable to get namespace: %w", err)
		}
		namespace = string(data)
	}

	// Create in-cluster Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get cluster config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("failed to create k8s client: %w", err)
	}

	// Fetch pod info
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get pod %s: %w", podName, err)
	}

	return pod.Spec.NodeName, nil
}

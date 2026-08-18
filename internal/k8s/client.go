package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Clients bundles the read-only Kubernetes clients the triage layer needs.
// Metrics is nil when the metrics API cannot be reached; callers must degrade.
type Clients struct {
	Kube    kubernetes.Interface
	Metrics metricsclientset.Interface
}

// ResolveKubeconfigPath applies precedence: explicit flag, then $KUBECONFIG,
// then ~/.kube/config.
func ResolveKubeconfigPath(flag string) string {
	if flag != "" {
		return flag
	}
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return env
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kube", "config")
}

// NewClients builds read-only clients. It prefers in-cluster config, then falls
// back to a kubeconfig file. A metrics client failure is non-fatal: Metrics is
// left nil so the caller degrades gracefully.
func NewClients(kubeconfig, kubeContext string) (*Clients, error) {
	cfg, err := restConfig(kubeconfig, kubeContext)
	if err != nil {
		return nil, err
	}
	kube, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("building kubernetes client: %w", err)
	}
	c := &Clients{Kube: kube}
	if m, err := metricsclientset.NewForConfig(cfg); err == nil {
		c.Metrics = m
	}
	return c, nil
}

func restConfig(kubeconfig, kubeContext string) (*rest.Config, error) {
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.ExplicitPath = ResolveKubeconfigPath(kubeconfig)
	overrides := &clientcmd.ConfigOverrides{}
	if kubeContext != "" {
		overrides.CurrentContext = kubeContext
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("resolving kubeconfig (no in-cluster config either): %w", err)
	}
	return cfg, nil
}

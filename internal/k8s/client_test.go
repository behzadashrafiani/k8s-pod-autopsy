package k8s

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveKubeconfigPath(t *testing.T) {
	t.Setenv("KUBECONFIG", "")
	if got := ResolveKubeconfigPath("/explicit/path"); got != "/explicit/path" {
		t.Fatalf("flag should win, got %q", got)
	}
	t.Setenv("KUBECONFIG", "/env/path")
	if got := ResolveKubeconfigPath(""); got != "/env/path" {
		t.Fatalf("env should win when no flag, got %q", got)
	}
	t.Setenv("KUBECONFIG", "")
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".kube", "config")
	if got := ResolveKubeconfigPath(""); got != want {
		t.Fatalf("default should be ~/.kube/config, got %q", got)
	}
}

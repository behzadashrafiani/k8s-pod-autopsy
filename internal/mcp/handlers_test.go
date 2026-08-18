package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/behzadashrafiani/k8s-pod-autopsy/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestAutopsyHandler_RendersMarkdown(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "web-1", Namespace: "demo"},
		Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{
			Name: "web", RestartCount: 2,
			LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 137, Reason: "OOMKilled"}},
		}}},
	}
	clients := &k8s.Clients{Kube: fake.NewSimpleClientset(pod)}
	h := makeAutopsyHandler(clients)
	_, out, err := h(context.Background(), nil, AutopsyInput{Namespace: "demo", PodName: "web-1"})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !strings.Contains(out.Markdown, "OOMKilled") {
		t.Fatalf("expected OOMKilled in markdown, got:\n%s", out.Markdown)
	}
}

func TestCrashLoopHandler_RendersMarkdown(t *testing.T) {
	bad := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "bad", Namespace: "demo"},
		Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{
			Name:  "c",
			State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
		}}},
	}
	clients := &k8s.Clients{Kube: fake.NewSimpleClientset(bad)}
	h := makeCrashLoopHandler(clients)
	_, out, err := h(context.Background(), nil, CrashLoopInput{Namespace: "demo"})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !strings.Contains(out.Markdown, "bad") || !strings.Contains(out.Markdown, "CrashLoopBackOff") {
		t.Fatalf("expected crashloop row, got:\n%s", out.Markdown)
	}
}

func TestNewServer_DoesNotPanic(t *testing.T) {
	clients := &k8s.Clients{Kube: fake.NewSimpleClientset()}
	if s := NewServer(clients); s == nil {
		t.Fatal("NewServer returned nil")
	}
}

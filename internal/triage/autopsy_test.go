package triage

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func podWith(phase corev1.PodPhase, init, app []corev1.ContainerStatus, spec []corev1.Container) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Spec:       corev1.PodSpec{Containers: spec},
		Status: corev1.PodStatus{
			Phase:                 phase,
			InitContainerStatuses: init,
			ContainerStatuses:     app,
		},
	}
}

func TestSelectCulprit_InitContainerCrashWinsOverApp(t *testing.T) {
	init := []corev1.ContainerStatus{{
		Name: "migrate",
		LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
			ExitCode: 1, Reason: "Error",
		}},
		RestartCount: 4,
	}}
	app := []corev1.ContainerStatus{{Name: "web", Ready: true}}
	f, sig, found := selectCulprit(podWith(corev1.PodPending, init, app, nil))
	if !found || sig != SignalCrashing || f.Role != RoleInit || f.Name != "migrate" {
		t.Fatalf("init crash should win: found=%v sig=%v f=%+v", found, sig, f)
	}
}

func TestSelectCulprit_WaitingImagePull(t *testing.T) {
	app := []corev1.ContainerStatus{{
		Name:  "web",
		State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff", Message: "back-off pulling image"}},
	}}
	f, sig, found := selectCulprit(podWith(corev1.PodPending, nil, app, nil))
	if !found || sig != SignalWaiting || f.WaitingReason != "ImagePullBackOff" {
		t.Fatalf("image pull should be Waiting signal: found=%v sig=%v f=%+v", found, sig, f)
	}
}

func TestSelectCulprit_OOMDetectedWithLimit(t *testing.T) {
	app := []corev1.ContainerStatus{{
		Name: "web",
		LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
			ExitCode: 137, Reason: "OOMKilled",
		}},
		RestartCount: 2,
	}}
	spec := []corev1.Container{{
		Name:      "web",
		Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("256Mi")}},
	}}
	f, sig, found := selectCulprit(podWith(corev1.PodRunning, nil, app, spec))
	if !found || sig != SignalOOMKilled || !f.HasMemLimit {
		t.Fatalf("OOM with limit expected: found=%v sig=%v f=%+v", found, sig, f)
	}
}

func TestSelectCulprit_Healthy(t *testing.T) {
	app := []corev1.ContainerStatus{{Name: "web", Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}}
	_, sig, found := selectCulprit(podWith(corev1.PodRunning, nil, app, nil))
	if found || sig != SignalHealthy {
		t.Fatalf("healthy pod should not be flagged: found=%v sig=%v", found, sig)
	}
}

// Edge fix #1: ephemeral (kubectl debug) containers must also be scanned.
func TestSelectCulprit_EphemeralContainerCrash(t *testing.T) {
	pod := podWith(corev1.PodRunning, nil,
		[]corev1.ContainerStatus{{Name: "web", Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}},
		nil)
	pod.Status.EphemeralContainerStatuses = []corev1.ContainerStatus{{
		Name: "debugger",
		LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
			ExitCode: 1, Reason: "Error",
		}},
		RestartCount: 1,
	}}
	f, sig, found := selectCulprit(pod)
	if !found || sig != SignalCrashing || f.Role != RoleEphemeral || f.Name != "debugger" {
		t.Fatalf("ephemeral crash should be detected: found=%v sig=%v f=%+v", found, sig, f)
	}
}

func TestAutopsyPod_NotFound(t *testing.T) {
	kube := fake.NewSimpleClientset()
	_, err := AutopsyPod(context.Background(), kube, nil, "ns", "missing")
	if err == nil || !contains(err.Error(), "not found") {
		t.Fatalf("expected not-found error, got %v", err)
	}
}

package triage

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestClassifyForScan(t *testing.T) {
	clb := podWith(corev1.PodRunning, nil, []corev1.ContainerStatus{{
		Name:  "c",
		State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
	}}, nil)
	if got, ok := classifyForScan(clb); !ok || got.Reason != ReasonCrashLoopBackOff {
		t.Fatalf("CrashLoopBackOff expected, got %+v ok=%v", got, ok)
	}

	oom := podWith(corev1.PodRunning, nil, []corev1.ContainerStatus{{
		Name:                 "c",
		LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled"}},
	}}, nil)
	if got, ok := classifyForScan(oom); !ok || got.Reason != ReasonOOMKilled {
		t.Fatalf("OOMKilled expected, got %+v ok=%v", got, ok)
	}

	high := podWith(corev1.PodRunning, nil, []corev1.ContainerStatus{{Name: "c", RestartCount: 7}}, nil)
	if got, ok := classifyForScan(high); !ok || got.Reason != ReasonHighRestart {
		t.Fatalf("HighRestart expected, got %+v ok=%v", got, ok)
	}

	healthy := podWith(corev1.PodRunning, nil, []corev1.ContainerStatus{{Name: "c", RestartCount: 1, Ready: true}}, nil)
	if _, ok := classifyForScan(healthy); ok {
		t.Fatalf("healthy pod should not be flagged")
	}
}

func TestFindCrashLooping_Lists(t *testing.T) {
	bad := podWith(corev1.PodRunning, nil, []corev1.ContainerStatus{{
		Name:  "c",
		State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
	}}, nil)
	bad.Name = "bad"
	good := podWith(corev1.PodRunning, nil, []corev1.ContainerStatus{{Name: "c", Ready: true}}, nil)
	good.Name = "good"
	kube := fake.NewSimpleClientset(bad, good)
	rep, err := FindCrashLooping(context.Background(), kube, "ns")
	if err != nil || len(rep.Pods) != 1 || rep.Pods[0].Name != "bad" {
		t.Fatalf("expected only 'bad' listed, got %+v err=%v", rep, err)
	}
	_ = metav1.NamespaceAll
}

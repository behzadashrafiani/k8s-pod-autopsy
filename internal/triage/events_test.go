package triage

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestFetchWarningEvents_FiltersNormal(t *testing.T) {
	kube := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "e1", Namespace: "ns"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "p"},
			Type:           corev1.EventTypeWarning,
			Reason:         "BackOff",
			Message:        "restarting failed container",
			Count:          3,
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "e2", Namespace: "ns"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "p"},
			Type:           corev1.EventTypeNormal,
			Reason:         "Pulled",
		},
	)
	got := fetchWarningEvents(context.Background(), kube, "ns", "p")
	if len(got) != 1 || got[0].Reason != "BackOff" || got[0].Count != 3 {
		t.Fatalf("expected only the Warning event, got %+v", got)
	}
}

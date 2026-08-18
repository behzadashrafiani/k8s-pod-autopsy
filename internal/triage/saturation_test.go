package triage

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"
)

// fakeMetricsWith returns a metrics fake clientset that answers a Get for the
// given PodMetrics. The metrics fake's object tracker registers PodMetrics
// under the guessed resource "podmetricses", but the typed Get uses resource
// "pods", so seeding via NewSimpleClientset(pm) is never found. Prepending a
// reactor on get/pods sidesteps that mismatch.
func fakeMetricsWith(pm *metricsv1beta1.PodMetrics) *metricsfake.Clientset {
	m := metricsfake.NewSimpleClientset()
	m.PrependReactor("get", "pods", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, pm, nil
	})
	return m
}

func TestPctOfLimit(t *testing.T) {
	used := resource.MustParse("950Mi")
	limit := resource.MustParse("1Gi")
	got := pctOfLimit(used, limit)
	if got < 90 || got > 95 {
		t.Fatalf("950Mi of 1Gi should be ~92%%, got %v", got)
	}
	none := resource.Quantity{}
	if pctOfLimit(used, none) != -1 {
		t.Fatalf("no limit should return -1")
	}
}

func TestCheckSaturation_NilMetrics(t *testing.T) {
	kube := fake.NewSimpleClientset()
	rep := CheckSaturation(context.Background(), kube, nil, "ns", "p")
	if rep.Unavailable == "" {
		t.Fatalf("nil metrics should set Unavailable, got %+v", rep)
	}
}

// Edge fix #2: metrics may lack CPU/Memory keys (new pod / lagging metrics).
// CheckSaturation must not panic and should still emit a row.
func TestCheckSaturation_MissingUsageKeys(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name: "web",
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("1Gi")},
			},
		}}},
	}
	kube := fake.NewSimpleClientset(pod)
	pm := &metricsv1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Containers: []metricsv1beta1.ContainerMetrics{{
			Name:  "web",
			Usage: corev1.ResourceList{}, // empty — no CPU/Memory keys yet
		}},
	}
	metrics := fakeMetricsWith(pm)
	rep := CheckSaturation(context.Background(), kube, metrics, "ns", "p")
	if rep.Unavailable != "" {
		t.Fatalf("should degrade to a partial row, not Unavailable: %+v", rep)
	}
	if len(rep.Containers) != 1 || rep.Containers[0].Name != "web" {
		t.Fatalf("expected one container row, got %+v", rep.Containers)
	}
}

func TestCheckSaturation_FlagsMemRisk(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name: "web",
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("1Gi")},
			},
		}}},
	}
	kube := fake.NewSimpleClientset(pod)
	pm := &metricsv1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns"},
		Containers: []metricsv1beta1.ContainerMetrics{{
			Name:  "web",
			Usage: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("950Mi")},
		}},
	}
	metrics := fakeMetricsWith(pm)
	rep := CheckSaturation(context.Background(), kube, metrics, "ns", "p")
	if len(rep.Containers) != 1 || !rep.Containers[0].MemRisk {
		t.Fatalf("950Mi of 1Gi should flag MemRisk, got %+v", rep.Containers)
	}
}

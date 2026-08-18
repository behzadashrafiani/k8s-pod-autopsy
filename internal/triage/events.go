package triage

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

// fetchWarningEvents returns only Warning-type events for the pod. On any error
// it returns an empty slice — events are best-effort context, never fatal.
func fetchWarningEvents(ctx context.Context, kube kubernetes.Interface, ns, pod string) []EventLine {
	sel := fields.AndSelectors(
		fields.OneTermEqualSelector("involvedObject.name", pod),
		fields.OneTermEqualSelector("involvedObject.kind", "Pod"),
	).String()
	list, err := kube.CoreV1().Events(ns).List(ctx, metav1.ListOptions{FieldSelector: sel})
	if err != nil {
		return nil
	}
	var out []EventLine
	for _, e := range list.Items {
		if e.Type != corev1.EventTypeWarning {
			continue
		}
		out = append(out, EventLine{Reason: e.Reason, Message: e.Message, Count: e.Count})
	}
	return out
}

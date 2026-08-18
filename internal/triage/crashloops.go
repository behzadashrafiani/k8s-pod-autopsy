package triage

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const crashLoopRestartThreshold = 5

// classifyForScan returns a CrashLoopPod if the pod is CrashLoopBackOff,
// OOMKilled, or exceeds the restart threshold. Init and app containers both count.
func classifyForScan(pod *corev1.Pod) (CrashLoopPod, bool) {
	all := append([]corev1.ContainerStatus{}, pod.Status.InitContainerStatuses...)
	all = append(all, pod.Status.ContainerStatuses...)
	for _, s := range all {
		base := CrashLoopPod{Namespace: pod.Namespace, Name: pod.Name, Container: s.Name, RestartCount: s.RestartCount}
		if w := s.State.Waiting; w != nil && w.Reason == "CrashLoopBackOff" {
			base.Reason = ReasonCrashLoopBackOff
			return base, true
		}
		if t := s.LastTerminationState.Terminated; t != nil && t.Reason == "OOMKilled" {
			base.Reason = ReasonOOMKilled
			return base, true
		}
		if s.RestartCount > crashLoopRestartThreshold {
			base.Reason = ReasonHighRestart
			return base, true
		}
	}
	return CrashLoopPod{}, false
}

// FindCrashLooping scans a namespace (or all namespaces when ns=="") for
// crashlooping / OOMed / high-restart pods.
func FindCrashLooping(ctx context.Context, kube kubernetes.Interface, ns string) (*CrashLoopReport, error) {
	scope := ns
	if scope == "" {
		scope = metav1.NamespaceAll
	}
	list, err := kube.CoreV1().Pods(scope).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	rep := &CrashLoopReport{Scanned: len(list.Items)}
	for i := range list.Items {
		if hit, ok := classifyForScan(&list.Items[i]); ok {
			rep.Pods = append(rep.Pods, hit)
		}
	}
	return rep, nil
}

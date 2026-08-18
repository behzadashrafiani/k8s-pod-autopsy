package triage

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

const autopsyRestartThreshold = 3

// AutopsyPod gathers pod state and returns an auto-routed report.
func AutopsyPod(ctx context.Context, kube kubernetes.Interface, metrics metricsclientset.Interface, ns, name string) (*PodReport, error) {
	pod, err := kube.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("pod %q not found in namespace %q", name, ns)
		}
		if apierrors.IsForbidden(err) {
			return nil, fmt.Errorf("insufficient permissions to read pod %q in namespace %q", name, ns)
		}
		return nil, err
	}

	rep := &PodReport{Namespace: ns, Name: name, Phase: string(pod.Status.Phase)}
	rep.Events = fetchWarningEvents(ctx, kube, ns, name)

	culprit, signal, found := selectCulprit(pod)
	rep.Signal = signal
	rep.Culprit = culprit

	evicted := hasEvictedEvent(rep.Events)

	switch signal {
	case SignalOOMKilled:
		rep.OOM = ptr(AnalyzeOOM(culprit.HasMemLimit, evicted))
		rep.Verdict = verdictFor(culprit, "OOMKilled — "+rep.OOM.Verdict)
		rep.Logs = gatherLogs(ctx, kube, ns, name, culprit, true)
	case SignalCrashing:
		rep.Verdict = verdictFor(culprit, "crashing, exit "+ExplainExitCode(culprit.ExitCode))
		rep.Logs = gatherLogs(ctx, kube, ns, name, culprit, true)
	case SignalWaiting:
		// No logs exist for a container that never started.
		rep.Verdict = verdictFor(culprit, culprit.WaitingReason+": "+culprit.WaitingMessage)
	case SignalPending:
		rep.Verdict = "Pod is Pending / unschedulable — see Warning events."
	case SignalSaturated:
		rep.Verdict = "Running but resource-saturated — see saturation table."
		rep.Saturation = CheckSaturation(ctx, kube, metrics, ns, name)
	default: // Healthy
		rep.Verdict = "No issues detected."
	}
	_ = found
	return rep, nil
}

// selectCulprit scans init containers first, then app containers, then
// ephemeral (debug) containers, and returns the first problem found with its
// signal. Returns SignalHealthy when none.
func selectCulprit(pod *corev1.Pod) (*ContainerFinding, Signal, bool) {
	specByName := map[string]corev1.Container{}
	for _, c := range pod.Spec.InitContainers {
		specByName[c.Name] = c
	}
	for _, c := range pod.Spec.Containers {
		specByName[c.Name] = c
	}

	scan := func(statuses []corev1.ContainerStatus, role ContainerRole) (*ContainerFinding, Signal, bool) {
		for _, s := range statuses {
			f := &ContainerFinding{Name: s.Name, Role: role, RestartCount: s.RestartCount}
			if spec, ok := specByName[s.Name]; ok {
				_, f.HasMemLimit = spec.Resources.Limits[corev1.ResourceMemory]
			}
			// 1 & 2: terminated with OOM or non-zero exit (via lastState).
			if t := s.LastTerminationState.Terminated; t != nil {
				f.ExitCode, f.ExitReason, f.Message = t.ExitCode, t.Reason, t.Message
				f.ExitPlain = ExplainExitCode(t.ExitCode)
				if t.Reason == "OOMKilled" {
					return f, SignalOOMKilled, true
				}
				if t.ExitCode != 0 {
					return f, SignalCrashing, true
				}
			}
			// Current terminated (rare: dead right now).
			if t := s.State.Terminated; t != nil && t.ExitCode != 0 {
				f.ExitCode, f.ExitReason, f.Message = t.ExitCode, t.Reason, t.Message
				f.ExitPlain = ExplainExitCode(t.ExitCode)
				if t.Reason == "OOMKilled" {
					return f, SignalOOMKilled, true
				}
				return f, SignalCrashing, true
			}
			// Waiting states.
			if w := s.State.Waiting; w != nil {
				if w.Reason == "CrashLoopBackOff" {
					return f, SignalCrashing, true
				}
				// Non-starting: image pull / config errors — no logs.
				f.WaitingReason, f.WaitingMessage = w.Reason, w.Message
				return f, SignalWaiting, true
			}
			// High restart even if currently running.
			if s.RestartCount > autopsyRestartThreshold {
				return f, SignalCrashing, true
			}
		}
		return nil, SignalHealthy, false
	}

	if f, sig, ok := scan(pod.Status.InitContainerStatuses, RoleInit); ok {
		return f, sig, true
	}
	if f, sig, ok := scan(pod.Status.ContainerStatuses, RoleApp); ok {
		return f, sig, true
	}
	if f, sig, ok := scan(pod.Status.EphemeralContainerStatuses, RoleEphemeral); ok {
		return f, sig, true
	}
	if pod.Status.Phase == corev1.PodPending {
		return nil, SignalPending, true
	}
	return nil, SignalHealthy, false
}

func gatherLogs(ctx context.Context, kube kubernetes.Interface, ns, pod string, c *ContainerFinding, wantPrevious bool) *LogSection {
	sec := &LogSection{Source: "previous"}
	raw, err := fetchContainerLog(ctx, kube, ns, pod, c.Name, wantPrevious)
	if err != nil {
		// No previous instance → fall back to current.
		raw, err = fetchContainerLog(ctx, kube, ns, pod, c.Name, false)
		if err != nil {
			sec.Unavailable = "logs unavailable: " + err.Error()
			return sec
		}
		sec.Source, sec.Fallback = "current", true
	}
	sec.Errors, sec.Lines = processLogs(raw)
	return sec
}

func hasEvictedEvent(events []EventLine) bool {
	for _, e := range events {
		if strings.EqualFold(e.Reason, "Evicted") {
			return true
		}
	}
	return false
}

func verdictFor(c *ContainerFinding, detail string) string {
	role := ""
	if c != nil && c.Role == RoleInit {
		role = " (init container)"
	} else if c != nil && c.Role == RoleEphemeral {
		role = " (ephemeral container)"
	}
	name := ""
	if c != nil {
		name = c.Name
	}
	return fmt.Sprintf("Container %q%s: %s", name, role, detail)
}

func ptr[T any](v T) *T { return &v }

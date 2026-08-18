package triage

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

const saturationRiskPct = 90.0

// pctOfLimit returns used/limit*100, or -1 when no limit is set.
func pctOfLimit(used, limit resource.Quantity) float64 {
	if limit.IsZero() {
		return -1
	}
	return float64(used.MilliValue()) / float64(limit.MilliValue()) * 100
}

// CheckSaturation compares live usage to requests/limits. Degrades to an
// Unavailable report when the metrics client is nil or the API errors.
func CheckSaturation(ctx context.Context, kube kubernetes.Interface, metrics metricsclientset.Interface, ns, name string) *SaturationReport {
	rep := &SaturationReport{Namespace: ns, Name: name}
	if metrics == nil {
		rep.Unavailable = "metrics unavailable (metrics-server not installed?)"
		return rep
	}
	pm, err := metrics.MetricsV1beta1().PodMetricses(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		rep.Unavailable = "metrics unavailable (metrics-server not installed?): " + err.Error()
		return rep
	}
	pod, err := kube.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		rep.Unavailable = "could not read pod spec for limits: " + err.Error()
		return rep
	}
	specByName := map[string]corev1.Container{}
	for _, c := range pod.Spec.Containers {
		specByName[c.Name] = c
	}
	for _, cm := range pm.Containers {
		// Edge fix #2: metrics may be lagging on a fresh pod — the Usage map can
		// be empty or missing CPU/Memory keys. Guard every lookup so we degrade
		// to "n/a" rather than reporting a misleading zero.
		cs := ContainerSaturation{
			Name:        cm.Name,
			CPUPctLimit: -1,
			MemPctLimit: -1,
		}
		cpuUsed, hasCPU := cm.Usage[corev1.ResourceCPU]
		memUsed, hasMem := cm.Usage[corev1.ResourceMemory]
		if hasCPU {
			cs.CPUUsed = cpuUsed.String()
		} else {
			cs.CPUUsed = "n/a"
		}
		if hasMem {
			cs.MemUsed = memUsed.String()
		} else {
			cs.MemUsed = "n/a"
		}
		if spec, ok := specByName[cm.Name]; ok {
			cpuLim := spec.Resources.Limits[corev1.ResourceCPU]
			memLim := spec.Resources.Limits[corev1.ResourceMemory]
			cpuReq := spec.Resources.Requests[corev1.ResourceCPU]
			memReq := spec.Resources.Requests[corev1.ResourceMemory]
			cs.CPULimit, cs.MemLimit = cpuLim.String(), memLim.String()
			cs.CPURequest, cs.MemRequest = cpuReq.String(), memReq.String()
			if hasCPU {
				cs.CPUPctLimit = pctOfLimit(cpuUsed, cpuLim)
				cs.CPUThrottleRisk = cs.CPUPctLimit >= saturationRiskPct
			}
			if hasMem {
				cs.MemPctLimit = pctOfLimit(memUsed, memLim)
				cs.MemRisk = cs.MemPctLimit >= saturationRiskPct
			}
		}
		rep.Containers = append(rep.Containers, cs)
	}
	return rep
}

package render

import (
	"strings"
	"testing"

	"github.com/behzadashrafiani/k8s-pod-autopsy/internal/triage"
)

func TestPodReport_OOMIncludesWhyAndPrevLogs(t *testing.T) {
	r := &triage.PodReport{
		Namespace: "ns", Name: "web-1", Phase: "Running", Signal: triage.SignalOOMKilled,
		Verdict: `Container "web" : OOMKilled`,
		Culprit: &triage.ContainerFinding{Name: "web", Role: triage.RoleApp, ExitCode: 137, ExitPlain: "137 (SIGKILL — OOM or forced kill)", HasMemLimit: true},
		OOM:     &triage.OOMAnalysis{HitOwnLimit: true, Verdict: "Container exceeded its own memory limit."},
		Logs:    &triage.LogSection{Source: "previous", Errors: []string{"panic: oom"}, Lines: []string{"panic: oom"}},
	}
	out := PodReport(r, PlainStyler())
	for _, want := range []string{"OOMKilled", "own memory limit", "previous", "panic: oom", "137"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestSaturation_FlagsRisk(t *testing.T) {
	r := &triage.SaturationReport{
		Namespace: "ns", Name: "web-1",
		Containers: []triage.ContainerSaturation{{Name: "web", MemUsed: "950Mi", MemLimit: "1Gi", MemPctLimit: 92, MemRisk: true}},
	}
	out := Saturation(r, PlainStyler())
	if !strings.Contains(out, "OOM risk") || !strings.Contains(out, "92") {
		t.Errorf("expected OOM risk callout, got:\n%s", out)
	}
}

func TestSaturation_Unavailable(t *testing.T) {
	out := Saturation(&triage.SaturationReport{Unavailable: "metrics unavailable"}, PlainStyler())
	if !strings.Contains(out, "metrics unavailable") {
		t.Errorf("expected degradation message, got:\n%s", out)
	}
}

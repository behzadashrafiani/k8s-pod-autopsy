package render

import (
	"fmt"
	"strings"

	"github.com/behzadashrafiani/k8s-pod-autopsy/internal/triage"
)

func PodReport(r *triage.PodReport, st Styler) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", st.Heading("# Pod Autopsy: "+r.Namespace+"/"+r.Name))
	fmt.Fprintf(&b, "**Verdict:** %s\n\n", st.Danger(r.Verdict))
	fmt.Fprintf(&b, "- Phase: %s\n- Signal: %s\n", r.Phase, r.Signal)
	if r.Culprit != nil {
		fmt.Fprintf(&b, "- Container: %s (%s), restarts: %d\n", r.Culprit.Name, r.Culprit.Role, r.Culprit.RestartCount)
		if r.Culprit.ExitPlain != "" {
			fmt.Fprintf(&b, "- Exit code: %s\n", r.Culprit.ExitPlain)
		}
		if r.Culprit.WaitingReason != "" {
			fmt.Fprintf(&b, "- Waiting: %s — %s\n", r.Culprit.WaitingReason, r.Culprit.WaitingMessage)
		}
	}
	if r.OOM != nil {
		fmt.Fprintf(&b, "\n## Why OOM\n%s\n", st.Danger(r.OOM.Verdict))
	}
	if r.Logs != nil {
		b.WriteString("\n## Logs (" + r.Logs.Source)
		if r.Logs.Fallback {
			b.WriteString(", fell back from previous")
		}
		b.WriteString(")\n")
		if r.Logs.Unavailable != "" {
			b.WriteString(st.Dim(r.Logs.Unavailable) + "\n")
		} else {
			if len(r.Logs.Errors) > 0 {
				b.WriteString("**Detected errors:**\n```\n" + strings.Join(r.Logs.Errors, "\n") + "\n```\n")
			}
			b.WriteString("```\n" + strings.Join(r.Logs.Lines, "\n") + "\n```\n")
		}
	}
	if r.Saturation != nil {
		b.WriteString("\n" + Saturation(r.Saturation, st))
	}
	if len(r.Events) > 0 {
		b.WriteString("\n## Warning Events\n")
		for _, e := range r.Events {
			fmt.Fprintf(&b, "- **%s** (×%d): %s\n", e.Reason, e.Count, e.Message)
		}
	}
	return b.String()
}

func Saturation(r *triage.SaturationReport, st Styler) string {
	var b strings.Builder
	b.WriteString(st.Heading("## Resource Saturation") + "\n")
	if r.Unavailable != "" {
		b.WriteString(st.Dim(r.Unavailable) + "\n")
		return b.String()
	}
	b.WriteString("\n| Container | CPU used/limit (%) | Mem used/limit (%) | Flag |\n")
	b.WriteString("|---|---|---|---|\n")
	for _, c := range r.Containers {
		flag := ""
		if c.MemRisk {
			flag = st.Danger("⚠ OOM risk (mem ≥90% of limit)")
		} else if c.CPUThrottleRisk {
			flag = "possible CPU throttling (snapshot)"
		}
		fmt.Fprintf(&b, "| %s | %s/%s (%s) | %s/%s (%s) | %s |\n",
			c.Name, c.CPUUsed, c.CPULimit, pctStr(c.CPUPctLimit),
			c.MemUsed, c.MemLimit, pctStr(c.MemPctLimit), flag)
	}
	return b.String()
}

func CrashLoops(r *triage.CrashLoopReport, st Styler) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\nScanned %d pods, %d flagged.\n\n", st.Heading("# CrashLooping Pods"), r.Scanned, len(r.Pods))
	if len(r.Pods) == 0 {
		b.WriteString("No crashlooping / OOMed / high-restart pods found.\n")
		return b.String()
	}
	b.WriteString("| Namespace | Pod | Container | Reason | Restarts |\n|---|---|---|---|---|\n")
	for _, p := range r.Pods {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %d |\n", p.Namespace, p.Name, p.Container, p.Reason, p.RestartCount)
	}
	return b.String()
}

func pctStr(v float64) string {
	if v < 0 {
		return "no limit"
	}
	return fmt.Sprintf("%.0f%%", v)
}

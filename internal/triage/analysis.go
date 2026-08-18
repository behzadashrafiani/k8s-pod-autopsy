package triage

import "fmt"

// ExplainExitCode maps a container exit code to a short plain-English cause.
func ExplainExitCode(code int32) string {
	switch code {
	case 0:
		return "0 (success)"
	case 1:
		return "1 (general application error)"
	case 2:
		return "2 (misuse / application error)"
	case 126:
		return "126 (container command not executable — permissions?)"
	case 127:
		return "127 (command not found — bad entrypoint/PATH)"
	case 139:
		return "139 (SIGSEGV — segfault)"
	case 143:
		return "143 (SIGTERM — often a failed liveness probe or graceful shutdown, not a true crash)"
	case 137:
		return "137 (SIGKILL — OOM or forced kill)"
	default:
		return fmt.Sprintf("%d (see logs)", code)
	}
}

// AnalyzeOOM explains WHY an OOM happened based on whether a memory limit was
// set and whether the pod was evicted under node pressure.
func AnalyzeOOM(hasMemLimit, evicted bool) OOMAnalysis {
	switch {
	case hasMemLimit:
		return OOMAnalysis{
			HitOwnLimit: true,
			Verdict:     "Container exceeded its own memory limit. Raise the limit or investigate a memory leak.",
		}
	default:
		return OOMAnalysis{
			NodePressure: true,
			Verdict:      "No memory limit set; killed under node memory pressure. Set a memory limit and check node capacity / noisy neighbors.",
		}
	}
}

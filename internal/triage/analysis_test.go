package triage

import "testing"

func TestExplainExitCode(t *testing.T) {
	cases := map[int32]string{
		137: "SIGKILL",
		143: "SIGTERM",
		139: "SIGSEGV",
		126: "not executable",
		127: "command not found",
	}
	for code, want := range cases {
		if got := ExplainExitCode(code); !contains(got, want) {
			t.Errorf("code %d: got %q, want it to mention %q", code, got, want)
		}
	}
	if got := ExplainExitCode(42); !contains(got, "42") {
		t.Errorf("unknown code should echo the number, got %q", got)
	}
}

func TestAnalyzeOOM(t *testing.T) {
	own := AnalyzeOOM(true, false)
	if !own.HitOwnLimit || own.NodePressure {
		t.Fatalf("limit-set OOM should be own-limit, got %+v", own)
	}
	node := AnalyzeOOM(false, false)
	if node.HitOwnLimit || !node.NodePressure {
		t.Fatalf("no-limit OOM should be node-pressure, got %+v", node)
	}
	if !contains(node.Verdict, "limit") {
		t.Errorf("node-pressure verdict should suggest setting a limit, got %q", node.Verdict)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
